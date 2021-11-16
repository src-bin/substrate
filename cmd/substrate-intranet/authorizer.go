package main

import (
	"context"
	"log"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
)

func authorizer(ctx context.Context, event *events.APIGatewayCustomAuthorizerRequestTypeRequest) (*events.APIGatewayCustomAuthorizerResponse, error) {

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	c, err := oauthoidc.NewClient(sess, event.StageVariables)
	if err != nil {
		return nil, err
	}

	u := &url.URL{
		Path:     event.Path,
		RawQuery: url.Values(event.MultiValueQueryStringParameters).Encode(),
	}
	next := u.String()
	u.Path = "/login"
	u.RawQuery = url.Values{"next": []string{next}}.Encode()
	context := map[string]interface{}{
		"Location": u.String(), // where API Gateway will send the browser when unauthorized
	}

	idToken := &oauthoidc.IDToken{}
	req := &http.Request{Header: http.Header{
		"Cookie": event.MultiValueHeaders["cookie"], // beware the case-sensitivity
	}}
	for _, cookie := range req.Cookies() {
		switch cookie.Name {
		case "a":
			context["AccessToken"] = cookie.Value
		case "id":
			if _, err := oauthoidc.ParseAndVerifyJWT(cookie.Value, c, idToken); err != nil {
				context["Error"] = err
				log.Print(err)
				idToken = &oauthoidc.IDToken{} // revert to zero-value and thus to denying access
				continue
			}
			if context["IDToken"], err = idToken.JSONString(); err != nil {
				return nil, err
			}
		}
	}

	effect := policies.Deny
	if idToken.Email != "" {
		roleName, err := c.RoleNameFromIdP(idToken.Email)
		if err == nil {
			context["RoleName"] = roleName
			effect = policies.Allow
		} else {
			context["Error"] = err
			log.Print(err)
		}
	}

	return &events.APIGatewayCustomAuthorizerResponse{
		Context: context,
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
