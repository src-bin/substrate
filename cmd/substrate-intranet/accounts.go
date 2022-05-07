package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/roles"
)

// TODO revamp accounts.html to bounce login requests through the logout page per <https://src-bin.slack.com/archives/C015H14T9UY/p1645052508548779>
//go:generate go run ../../tools/template/main.go -name accountsTemplate -package main accounts.html

func accountsHandler(ctx context.Context, cfg *awscfg.Config, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	c, err := oauthoidc.NewClient(sess, event.StageVariables)
	if err != nil {
		return nil, err
	}
	_ = c

	accountId := event.QueryStringParameters["number"]
	roleName := event.QueryStringParameters["role"]
	if accountId != "" && roleName != "" {

		// We have to start from the user's configured starting point so that
		// all questions of authorization are deferred to AWS.
		cfg, err = cfg.AssumeRole(
			ctx,
			event.RequestContext.AccountID,
			event.RequestContext.Authorizer[authorizerutil.RoleName].(string),
			fmt.Sprint(event.RequestContext.Authorizer["principalId"]),
		)

		roleArn := roles.Arn(accountId, roleName)
		cfg.Telemetry().SetFinalAccountId(accountId)
		cfg.Telemetry().SetFinalRoleName(roleArn)
		cfg, err = cfg.AssumeRole(
			ctx,
			accountId,
			roleName,
			fmt.Sprint(event.RequestContext.Authorizer["principalId"]),
		)
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}
		credentials, err := cfg.Retrieve(ctx)
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}

		consoleSigninURL, err := federation.ConsoleSigninURL(credentials, "")
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}

		return &events.APIGatewayProxyResponse{
			Body: fmt.Sprintf("redirecting to %s", consoleSigninURL),
			Headers: map[string]string{
				"Content-Type": "text/plain",
				"Location":     consoleSigninURL,
			},
			StatusCode: http.StatusFound,
		}, nil
	}

	sess, err = awssessions.InManagementAccount(roles.OrganizationReader, awssessions.Config{})
	if err != nil {
		return nil, err
	}
	svc := organizations.New(sess)

	adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(svc)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}

	body, err := lambdautil.RenderHTML(accountsTemplate(), struct {
		AdminAccounts, ServiceAccounts                                 []*awsorgs.Account
		AuditAccount, DeployAccount, ManagementAccount, NetworkAccount *awsorgs.Account
		RoleName                                                       string
	}{
		adminAccounts, serviceAccounts,
		auditAccount, deployAccount, managementAccount, networkAccount,
		event.RequestContext.Authorizer[authorizerutil.RoleName].(string),
	})
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil

}

func init() {
	handlers["/accounts"] = accountsHandler
}
