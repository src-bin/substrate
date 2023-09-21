package main

import (
	"context"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/ui"
)

func authorizer2(
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
) func(
	context.Context,
	*events.APIGatewayV2CustomAuthorizerV2Request,
) (*events.APIGatewayV2CustomAuthorizerIAMPolicyResponse, error) {
	return func(
		ctx context.Context,
		event *events.APIGatewayV2CustomAuthorizerV2Request,
	) (*events.APIGatewayV2CustomAuthorizerIAMPolicyResponse, error) {
		ctx = contextutil.WithValues(ctx, "substrate-intranet", "authorizer", "")

		u := &url.URL{
			Path:     event.RawPath,
			RawQuery: event.RawQueryString,
		}
		next := u.String()
		u.Path = "/login"
		u.RawQuery = url.Values{"next": []string{next}}.Encode()
		authContext := map[string]interface{}{
			"Location": u.String(), // where API Gateway will send the browser when unauthorized
		}

		idToken := &oauthoidc.IDToken{}
		for _, cookie := range lambdautil.Cookies2(event.Cookies) {
			switch cookie.Name {
			case "a":
				authContext[authorizerutil.AccessToken] = cookie.Value
			case "id":
				_, err := oauthoidc.ParseAndVerifyJWT(cookie.Value, oc, idToken)
				if err != nil {
					authContext[authorizerutil.Error] = err
					ui.PrintWithCaller(err)
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
			authContext[authorizerutil.PrincipalId] = idToken.Email // would be overkill except see the comment on PrincipalID below
			roleName, err := oc.WithAccessToken(authContext[authorizerutil.AccessToken].(string)).RoleNameFromIdP(idToken.Email)
			if err == nil {
				authContext[authorizerutil.RoleName] = roleName
				effect = policies.Allow
			} else {
				authContext[authorizerutil.Error] = err
				ui.PrintWithCaller(err)
			}
		}

		return &events.APIGatewayV2CustomAuthorizerIAMPolicyResponse{
			Context: authContext,
			PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
				Statement: []events.IAMPolicyStatement{{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect.String(),
					Resource: []string{event.RouteArn},
				}},
				Version: "2012-10-17",
			},
			PrincipalID: idToken.Email, // this is bizarrely not exposed to the Lambda function target so it's useless (but still correct)
		}, nil
	}
}
