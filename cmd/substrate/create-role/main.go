package createrole

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
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
		ui.Print("                                   [-domain <domain>][...] [-environment <environment>][...] [-quality <quality>][...]")
		ui.Print("                                   -number <number>[...]")
		ui.Print("       [assume-role policy flags]: [-humans] [-aws-service <aws-service-url>] [-github-actions <org/repo>] [-assume-role-policy <filename>]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *githubActions != "" && !strings.Contains(*githubActions, "/") {
		ui.Fatal(`-github-actions "..." must contain a '/'`)
	}
	if *quiet {
		ui.Quiet()
	}
	if *roleName == "" {
		ui.Fatal(`-role "..." is required`)
	}

	adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	ui.Must(err)
	_ = auditAccount                                                                                                  // TODO should we support adding roles in the audit account?
	_, _, _, _, _, _ = adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount // TODO remove and rectify

	// Construct the assume-role policy for this role as it will be created in
	// various accounts later. This governs who may assume this role.
	// TODO accommodate multiple invocations with subtly different options.
	policy := &policies.Document{}
	if *humans {
		ui.Printf("allowing humans to assume this role via your IdP")
		canned, err := admin.CannedPrincipals(ctx, cfg, false)
		ui.Must(err)
		principals := &policies.Principal{AWS: []string{}}

		// Create a role by the same name in admin accounts that can't
		// (necessarily) do the same things; its purpose is to be allowed
		// (later) to assume the roles created in the various other accounts.
		ui.Spinf("finding or creating the %s role in your admin account(s)", *roleName)
		for _, account := range adminAccounts {
			// TODO create a role in every admin account that's allowed to assume roles, use Credential/Instance Factory, etc.
			principals.AWS = append(principals.AWS, roles.ARN(aws.ToString(account.Id), *roleName))
		}
		ui.Stop("ok")

		policy = policies.Merge(
			policy,
			policies.AssumeRolePolicyDocument(canned.AdminRolePrincipals), // Administrator can do anything, after all
			policies.AssumeRolePolicyDocument(principals),
		)
	}
	if len(*awsServices) > 0 {
		ui.Printf("allowing %s to assume this role", strings.Join(*awsServices, ", "))
		policy = policies.Merge(
			policy,
			policies.AssumeRolePolicyDocument(&policies.Principal{
				Service: jsonutil.StringSlice(*awsServices),
			}),
		)
	}
	if *githubActions != "" {
		ui.Printf("allowing GitHub Actions to assume this role on behalf of the %s repository", *githubActions)
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
	var filePolicy policies.Document
	if err := jsonutil.Read(*assumeRolePolicy, &filePolicy); err == nil {
		ui.Printf("reading additional assume-role policy statements from %s", *assumeRolePolicy)
		policy = policies.Merge(policy, &filePolicy)
	} else if !errors.Is(err, fs.ErrNotExist) {
		ui.Fatal(err)
	}
	log.Print(jsonutil.MustString(policy))

	// Create this role in every special or service account selected by the
	// given options. Tag everything so thoroughly that `substrate roles` can
	// reconstruct this an every other create-role invocation.
	if *management {
		ui.Spinf("finding or creating the %s role in your management account", *roleName)
		// TODO create the role in aws.ToString(managementAccount.Id))
		ui.Stop("ok")
	}
	for _, special := range *specials {
		if special == "deploy" {
			ui.Spinf("finding or creating the %s role in your deploy account", *roleName)
			// TODO create the role in aws.ToString(deployAccount.Id))
			ui.Stop("ok")
		} else if special == "network" {
			ui.Spinf("finding or creating the %s role in your network account", *roleName)
			// TODO create the role in aws.ToString(networkAccount.Id))
			ui.Stop("ok")
		}
	}
	match := func(account *awsorgs.Account) bool {
		if len(*domains) != 0 && !contains(*domains, account.Tags[tagging.Domain]) {
			return false
		}
		if len(*environments) != 0 && !contains(*environments, account.Tags[tagging.Environment]) {
			return false
		}
		if len(*qualities) != 0 && !contains(*qualities, account.Tags[tagging.Quality]) {
			return false
		}
		return true
	}
	for _, account := range serviceAccounts {
		if match(account) {
			ui.Spinf("finding or creating the %s role in %s %+v", *roleName, aws.ToString(account.Id), account.Tags) // TODO better serialization of account
			// TODO create the role in aws.ToString(account.Id))
			ui.Stop("ok")
		}
	}
	if len(*numbers) > 0 {
		for _, number := range *numbers {
			ui.Spinf("finding or creating the %s role in account number %s", *roleName, number)
			// TODO create the role in number
			ui.Stop("ok")
		}
	}

	// TODO make -administrator and -auditor canned attached policies, too, plus -policy to attach a policy everywhere (except, possibly, the admin account)

	// TODO tag the roles to make sure we can find them later

	// TODO tag the roles abstractly enough that we'll be able to expand to cover newly created accounts that match their parameters at creation time

}

func contains(ss []string, s string) bool {
	for i := 0; i < len(ss); i++ {
		if ss[i] == s {
			return true
		}
	}
	return false
}
