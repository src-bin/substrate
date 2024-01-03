package accounts

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/roles"
)

func JavaScript(
	context.Context,
	*awscfg.Config,
	*oauthoidc.Client,
	*events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	return lambdautil.Static("application/javascript; charset=utf-8", javascript)
}

func Main(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	var err error

	accountId := event.QueryStringParameters["number"]
	roleName := event.QueryStringParameters["role"]
	if accountId != "" && roleName != "" {
		var cfg12h, credsCfg, credsCfg12h, userCfg *awscfg.Config

		// We have to start from the user's configured starting point so that
		// all questions of authorization are deferred to AWS.
		if userCfg, err = cfg.AssumeRole(
			ctx,
			event.RequestContext.AccountID,
			fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.RoleName]),
			time.Hour,
		); err != nil {
			return lambdautil.ErrorResponse2(err)
		}

		roleArn := roles.ARN(accountId, roleName)
		cfg.Telemetry().SetFinalAccountId(accountId)
		cfg.Telemetry().SetFinalRoleName(roleArn)

		// First, assume the role directly to ensure it's really authorized.
		if credsCfg, err = userCfg.AssumeRole(ctx, accountId, roleName, time.Hour); err != nil {
			return lambdautil.ErrorResponse2(err)
		}

		// If that worked, try to assume the role from an IAM user to get
		// 12-hour credentials.
		if cfg12h, err = awsiam.AllDayConfig(ctx, cfg); err != nil {
			return lambdautil.ErrorResponse2(err)
		}
		if credsCfg12h, err = cfg12h.AssumeRole(ctx, accountId, roleName, 12*time.Hour); err != nil {
			log.Print(err) // continue because this is optional
		}

		// Finally, fetch credentials. 12-hour if we can, 1-hour if we have to.
		var creds aws.Credentials
		if credsCfg12h != nil {
			creds, err = credsCfg12h.Retrieve(ctx)
		} else {
			creds, err = credsCfg.Retrieve(ctx)
		}
		if err != nil {
			return lambdautil.ErrorResponse2(err)
		}

		var destination string // empty will land on the AWS Console homepage
		if next := event.QueryStringParameters["next"]; next != "" {
			if u, err := url.Parse(next); err == nil {
				if strings.HasSuffix(u.Host, "console.aws.amazon.com") { // don't be an open redirect
					destination = next
				}
			}
		}

		consoleSigninURL, err := federation.ConsoleSigninURL2(
			creds,
			destination,
			event,
		)
		if err != nil {
			return lambdautil.ErrorResponse2(err)
		}

		return &events.APIGatewayV2HTTPResponse{
			Body: fmt.Sprintf("redirecting to %s", consoleSigninURL),
			Headers: map[string]string{
				"Content-Type":                   "text/plain",
				"Location":                       consoleSigninURL,
				"X-Substrate-Credentials-Expire": creds.Expires.Format(time.RFC3339),
			},
			StatusCode: http.StatusFound,
		}, nil
	}

	if cfg, err = cfg.OrganizationReader(ctx); err != nil {
		return lambdautil.ErrorResponse2(err)
	}
	adminAccounts, serviceAccounts, substrateAccount, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		return lambdautil.ErrorResponse2(err)
	}

	body, err := lambdautil.RenderHTML(html, struct {
		AdminAccounts, ServiceAccounts                                 []*awsorgs.Account
		SubstrateAccount                                               *awsorgs.Account
		AuditAccount, DeployAccount, ManagementAccount, NetworkAccount *awsorgs.Account
		RoleName                                                       string
	}{
		adminAccounts, serviceAccounts,
		substrateAccount,
		auditAccount, deployAccount, managementAccount, networkAccount,
		fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.RoleName]),
	})
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		StatusCode: http.StatusOK,
	}, nil

}

//go:embed accounts.html
var html string

//go:embed accounts.js
var javascript string
