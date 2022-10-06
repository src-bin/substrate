package main

import (
	"context"
	"log"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
)

func authorizer(
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
) func(
	context.Context,
	*events.APIGatewayCustomAuthorizerRequestTypeRequest,
) (*events.APIGatewayCustomAuthorizerResponse, error) {
	return func(
		ctx context.Context,
		event *events.APIGatewayCustomAuthorizerRequestTypeRequest,
	) (*events.APIGatewayCustomAuthorizerResponse, error) {
		ctx = contextutil.WithValues(ctx, "substrate-intranet", "authorizer", "")

		u := &url.URL{
			Path:     event.Path,
			RawQuery: url.Values(event.MultiValueQueryStringParameters).Encode(),
		}
		next := u.String()
		u.Path = "/login"
		u.RawQuery = url.Values{"next": []string{next}}.Encode()
		authContext := map[string]interface{}{
			"Location": u.String(), // where API Gateway will send the browser when unauthorized
		}

		// For performance testing purposes, it was at one time desirable to
		// short-circuit the rest of this function, which showed there was
		// easily half a second of latency to be saved. This is being committed
		// commented out and then promptly removed just so there's a record of
		// the experiment and how to do it, should it ever need to be repeated.
		/*
			if event.Headers["X-Forwarded-For"] == "67.180.198.56" { // rcrowley at home on 2022-10-05
				authContext[authorizerutil.RoleName] = roles.Administrator
				return &events.APIGatewayCustomAuthorizerResponse{
					Context: authContext,
					PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
						Statement: []events.IAMPolicyStatement{{
							Action:   []string{"execute-api:Invoke"},
							Effect:   string(policies.Allow),
							Resource: []string{event.MethodArn},
						}},
						Version: "2012-10-17",
					},
					PrincipalID: "rcrowley@src-bin.com",
				}, nil
			}
		*/

		idToken := &oauthoidc.IDToken{}
		req := &http.Request{Header: http.Header{
			"Cookie": event.MultiValueHeaders["cookie"], // beware the case-sensitivity
		}}
		for _, cookie := range req.Cookies() {
			switch cookie.Name {
			case "a":
				authContext[authorizerutil.AccessToken] = cookie.Value
			case "id":
				_, err := oauthoidc.ParseAndVerifyJWT(cookie.Value, oc, idToken)
				if err != nil {
					authContext[authorizerutil.Error] = err
					log.Print(err)
					idToken = &oauthoidc.IDToken{} // revert to zero-value and thus to denying access
					continue
				}
				ctx = context.WithValue(ctx, contextutil.Username, idToken.Email) // not that we need this (yet)
				if authContext[authorizerutil.IDToken], err = idToken.JSONString(); err != nil {
					return nil, err
				}
			}
		}

		effect := policies.Deny
		if idToken.Email != "" {
			oc.AccessToken = authContext[authorizerutil.AccessToken].(string)
			roleName, err := oc.RoleNameFromIdP(idToken.Email)
			if err == nil {
				authContext[authorizerutil.RoleName] = roleName
				effect = policies.Allow
			} else {
				authContext[authorizerutil.Error] = err
				log.Print(err)
			}
		}

		return &events.APIGatewayCustomAuthorizerResponse{
			Context: authContext,
			PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
				Statement: []events.IAMPolicyStatement{{
					Action:   []string{"execute-api:Invoke"},
					Effect:   string(effect),
					Resource: []string{event.MethodArn},
				}},
				Version: "2012-10-17",
			},
			PrincipalID: idToken.Email,
		}, nil
	}
}
