package roles

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatText) // default to undocumented special value
	flag.Usage = func() {
		ui.Print("Usage: substrate accounts [-format <format>]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	versionutil.WarnDowngrade(ctx, cfg)

	ui.Spin("inspecting all the roles in all your AWS accounts")
	allAccounts, err := cfg.ListAccounts(ctx)
	ui.Must(err)
	var (
		roleNames []string
		tree      = make(map[string][]treeNode)
	)
	for _, account := range allAccounts {

		// We can't assume an Administrator-like role in the audit account and
		// we wouldn't find anything useful there if we did so don't bother.
		if account.Tags[tagging.SubstrateSpecialAccount] == naming.Audit {
			continue
		}

		accountCfg := awscfg.Must(account.Config(ctx, cfg, account.AdministratorRoleName(), time.Hour))
		roles, err := awsiam.ListRoles(ctx, accountCfg)
		ui.Must(err)
		for _, role := range roles {
			if role.Tags[tagging.Manager] != tagging.Substrate {
				continue
			}
			if role.Tags[tagging.SubstrateAccountSelectors] == "" {
				continue
			}
			if _, ok := tree[role.Name]; !ok {
				roleNames = append(roleNames, role.Name)
			}
			tree[role.Name] = append(
				tree[role.Name],
				treeNode{account, role},
			)
		}
	}
	sort.Strings(roleNames) // so that eventual output is stable
	ui.Stop("ok")

	// Needed later but no need to parse it on every loop.
	u, err := url.Parse(awsiam.GitHubActionsOAuthOIDCURL)
	ui.Must(err)

	selections := make(map[string]*accounts.Selection)
	for _, roleName := range roleNames {
		selection := &accounts.Selection{}
		selections[roleName] = selection
		for _, tn := range tree[roleName] {
			account := tn.Account
			role := tn.Role

			// Derive the account selection flags from the selectors stored
			// in the SubstrateAccountSelectors tag on the role.
			selectors := strings.Split(role.Tags[tagging.SubstrateAccountSelectors], " ")
			for _, selector := range selectors {
				switch selector {
				case "all-domains":
					selection.AllDomains = true
				case "domain":
					selection.Domains = append(selection.Domains, account.Tags[tagging.Domain])
				case "all-environments":
					selection.AllEnvironments = true
				case "environment":
					environment := account.Tags[tagging.Environment]
					if naming.Index(selection.Environments, environment) < 0 {
						selection.Environments = append(selection.Environments, environment)
					}
				case "all-qualities":
					selection.AllQualities = true
				case "quality":
					quality := account.Tags[tagging.Quality]
					if naming.Index(selection.Qualities, quality) < 0 {
						selection.Qualities = append(selection.Qualities, quality)
					}
				case "admin":
					selection.Admin = true
				case "management":
					selection.Management = true
				case "special":
					selection.Specials = append(selection.Specials, account.Tags[tagging.SubstrateSpecialAccount])
				case "number":
					selection.Numbers = append(selection.Numbers, aws.ToString(account.Id))
				default:
					ui.Printf("unknown account selector %q", selector)
				}
			}
			sort.Strings(selection.Domains)
			environments, err := naming.Environments()
			ui.Must(err)
			naming.IndexedSort(selection.Environments, environments)
			qualities, err := naming.Qualities()
			ui.Must(err)
			naming.IndexedSort(selection.Qualities, qualities)

			// Derive most assume-role policy flags from the statements in the
			// assume-role policy.
			for _, statement := range role.AssumeRolePolicy.Statement {

				// -humans
				var credentialFactory, ec2, intranet bool
				for _, arn := range statement.Principal.AWS {
					if strings.HasSuffix(arn, fmt.Sprintf(":user/%s", users.CredentialFactory)) {
						credentialFactory = true
					}
					if strings.HasSuffix(arn, fmt.Sprintf(":role/%s", roles.Intranet)) {
						intranet = true
					}
				}
				for _, service := range statement.Principal.Service {
					if service == "ec2.amazonaws.com" {
						ec2 = true
					}
				}
				if credentialFactory && ec2 && intranet {
					log.Print("-humans") // XXX
				}

				// -aws-service "..."
				for _, service := range statement.Principal.Service {
					log.Printf("-aws-service %q", service) // XXX
				}

				// -github-actions "..."
				if len(statement.Principal.Federated) == 1 && strings.HasSuffix(statement.Principal.Federated[0], fmt.Sprintf("/%s", u.Host)) {
					for operator, predicates := range statement.Condition {
						if operator != "StringEquals" {
							continue
						}
						for key, values := range predicates {
							if key != fmt.Sprintf("%s:sub", u.Host) {
								continue
							}
							for _, value := range values {
								var repo string
								if _, err := fmt.Sscanf(value, "repo:%s:*", &repo); err != nil {
									continue
								}
								log.Printf("-github-actions %q", repo) // XXX
							}
						}
					}
				}

			}

			// Derive the -assume-role-policy flag from the
			// SubstrateAssumeRolePolicyFilenames tag, if present.
			filenames := strings.Split(role.Tags[tagging.SubstrateAssumeRolePolicyFilenames], " ")
			for _, filename := range filenames {
				log.Printf("-assume-role-policy %q", filename) // XXX
			}

		}
	}

	switch format.String() {

	case cmdutil.SerializationFormatJSON:
		log.Print(jsonutil.MustString(selections)) // TODO a real JSON output, not this stupid one that just satisfies that selections is used

	case cmdutil.SerializationFormatShell:
		for _, roleName := range roleNames {
			selection := selections[roleName]
			log.Printf("roleName: %s selection: %+v", roleName, selection)
			// TODO stringify selection into command-line arguments
			// TODO stringify assume-role policy detections into command-line arguments
		}

	case cmdutil.SerializationFormatText:
		for _, roleName := range roleNames {
			fmt.Println(roleName) // TODO include selectors, account numbers, maybe flagging whether it's outdated (i.e. new accounts with the domain), etc.
		}

	default:
		ui.Fatalf("-format %q not supported", format)
	}
}

type treeNode struct {
	Account *awsorgs.Account
	Role    *awsiam.Role
}
