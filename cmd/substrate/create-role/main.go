package createrole

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	// TODO do we want this or is it implied by -humans? admin := flag.Bool("admin", false, `shorthand for -domain "admin" -environment "admin"`)
	domains := cmdutil.StringSliceFlag("domain", "only create this role in AWS accounts in this domain (may be repeated; if unspecified, create the role in all domains)")
	environments := cmdutil.StringSliceFlag("environment", "only create this role in AWS accounts in this environment (may be repeated; if unspecified, create the role in all environments)")
	qualities := cmdutil.StringSliceFlag("quality", "only create this role in AWS accounts of this quality (may be repeated; if unspecified, create the role for all qualities)") // TODO do we need to do anything to cover the special case of only having one quality which may be omitted in single-account selection contexts?
	specials := cmdutil.StringSliceFlag("special", `create this role in a special AWS account (may be repeated; "deploy" and/or "network")`)
	management := flag.Bool("management", false, "create this role in the organization's management AWS account")
	numbers := cmdutil.StringSliceFlag("number", "create this role in a specific AWS account, by 12-digit account number (may be repeated)")
	roleName := flag.String("role", "", "name of the IAM role to create")
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	humans := flag.Bool("humans", false, "allow humans with this role set in your IdP to assume this role via the Credential Factory")
	awsServices := cmdutil.StringSliceFlag("aws-service", `allow an AWS service (by URL; e.g. "ec2.amazonaws.com") to assume role (may be repeated)`)
	githubActions := flag.String("github-actions", "", `allow GitHub Actions to assume this role in the context of the given GitHub organization and repository (separated by a literal '/')`)
	assumeRolePolicy := flag.String("assume-role-policy", "", "filename containing an assume-role policy to be merged into this role's final assume-role policy")
	flag.Usage = func() {
		ui.Print("Usage: substrate create-role [account selection flags] -role <role> [assume-role policy flags] [-quiet]")
		ui.Print("       [account selection flags]:  -management|-special <special>")
		ui.Print("                                   -admin [-quality <quality>]")
		ui.Print("                                   [-domain <domain>][...] [-environment <environment>][...] [-quality <quality>][...]")
		ui.Print("                                   -number <number>[...]")
		ui.Print("       [assume-role policy flags]: [-humans] [-aws-service <aws-service-url>] [-github-actions <org/repo>] [-assume-role-policy <filename>]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	/*
		if *admin {
			domains.Set("admin")
			environments.Set("admin")
		}
	*/
	if *githubActions != "" && !strings.Contains(*githubActions, "/") {
		ui.Fatal(`-github-actions "..." must contain a '/'`)
	}
	if *quiet {
		ui.Quiet()
	}
	if *roleName == "" {
		ui.Fatal(`-role "..." is required`)
	}

	var accountIds []string // accounts where we'll create the role

	adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	ui.Must(err)
	_, _, _, _, _, _ = adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount

	ui.Printf("domains: %+v", *domains)
	// TODO
	ui.Printf("environments: %+v", *environments)
	// TODO
	ui.Printf("qualities: %+v", *qualities)
	// TODO

	ui.Printf("management: %+v", *management)
	if *management {
		accountIds = append(accountIds, aws.ToString(managementAccount.Id))
	}
	ui.Printf("specials: %+v", *specials)
	for _, special := range *specials {
		if special == "deploy" {
			accountIds = append(accountIds, aws.ToString(deployAccount.Id))
		} else if special == "network" {
			accountIds = append(accountIds, aws.ToString(networkAccount.Id))
		}
	}

	ui.Printf("numbers: %+v", *numbers)
	accountIds = append(accountIds, *numbers...)

	ui.Printf("accountIds: %+v", accountIds)

	ui.Printf("roleName: %+v", *roleName)

	canned, err := admin.CannedAssumeRolePolicyDocuments(ctx, cfg, false)
	ui.Must(err)

	policy := &policies.Document{}

	ui.Printf("humans: %+v", *humans)
	policy = policies.Merge(
		policy,
		canned.AdminRolePrincipals, // TODO probably not quite what we want; should be CredentialFactory, Intranet, etc.
	)
	ui.Printf("awsServices: %+v", *awsServices)
	policy = policies.Merge(
		policy,
		policies.AssumeRolePolicyDocument(&policies.Principal{Service: jsonutil.StringSlice(*awsServices)}),
	)
	ui.Printf("githubActions: %+v", *githubActions)
	if *githubActions != "" {
		policy = policies.Merge(
			policy,
			&policies.Document{
				Statement: []policies.Statement{{
					Action: []string{"sts:AssumeRoleWithWebIdentity"},
					Condition: policies.Condition{"StringEquals": {
						"token.actions.githubusercontent.com:sub": fmt.Sprintf("repo:%s:*", *githubActions),
					}},
					Principal: &policies.Principal{
						Federated: []string{"TODO"}, // TODO create an OAuth OIDC provider per this Terraform copy-pasta
						/*
						   # <https://github.com/aws-actions/configure-aws-credentials>
						   resource "aws_iam_openid_connect_provider" "github" {
						     client_id_list  = ["sts.amazonaws.com"]
						     thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
						     url             = "https://token.actions.githubusercontent.com"
						   }
						*/
					},
				}},
			},
		)
	}

	ui.Printf("assumeRolePolicy: %+v", *assumeRolePolicy)
	var filePolicy policies.Document
	if err := jsonutil.Read(*assumeRolePolicy, &filePolicy); err != nil && !errors.Is(err, fs.ErrNotExist) {
		ui.Fatal(err)
	}

	ui.Print(jsonutil.MustString(policy))
}
