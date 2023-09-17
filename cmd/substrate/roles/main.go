package roles

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
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

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	format := cmdutil.SerializationFormatFlag(
		cmdutil.SerializationFormatText,
		`output format - "text" for human-readable plaintext, "json" for and tags, or "shell" for a shell program that will update all your AWS accounts`,
	)
	flag.Usage = func() {
		ui.Print("Usage: substrate accounts [-format <format>]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	versionutil.WarnDowngrade(ctx, cfg)

	// Gather up all the Substrate-managed roles from all the AWS accounts in
	// the whole organization so we can collate them.
	ui.Spin("inspecting all the roles in all your AWS accounts")
	allAccounts, err := cfg.ListAccounts(ctx)
	ui.Must(err)
	var (
		mu        sync.Mutex
		roleNames []string
		tree      = make(map[string][]treeNode)
		wg        sync.WaitGroup
	)
	for _, account := range allAccounts {
		wg.Add(1)
		go func(account *awscfg.Account) {
			defer wg.Done()

			// We can't assume an Administrator-like role in the audit account and
			// we wouldn't find anything useful there if we did so don't bother.
			if account.Tags[tagging.SubstrateSpecialAccount] == naming.Audit {
				return
			}

			accountCfg := awscfg.Must(account.Config(ctx, cfg, account.AdministratorRoleName(), time.Hour))
			roles, err := awsiam.ListRoles(ctx, accountCfg)
			ui.Must(err)
			for _, role := range roles { // TODO could possibly do this loop concurrently, too
				if role.Tags[tagging.Manager] != tagging.Substrate {
					continue
				}
				if role.Tags[tagging.SubstrateAccountSelectors] == "" {
					continue
				}
				arns, err := awsiam.ListAttachedRolePolicies(ctx, accountCfg, role.Name)
				ui.Must(err)
				mu.Lock()
				if _, ok := tree[role.Name]; !ok {
					roleNames = append(roleNames, role.Name)
				}
				tree[role.Name] = append(
					tree[role.Name],
					treeNode{
						Account:    account,
						PolicyARNs: arns,
						Role:       role,
					},
				)
				mu.Unlock()
			}
		}(account)
	}
	wg.Wait()
	sort.Strings(roleNames) // so that all output formats are stable
	for _, treeNodes := range tree {
		sort.Slice(treeNodes, func(i, j int) bool {
			return treeNodes[i].Role.ARN < treeNodes[j].Role.ARN
		}) // so that role ARNs in the text output are stable
	}
	ui.Stop("ok")

	// Needed later but no need to parse it on every loop.
	u, err := url.Parse(awsiam.GitHubActionsOAuthOIDCURL)
	ui.Must(err)

	// Get the Substrate account number for when we need it later to detect
	// -humans from the statements in an assume-role policy.
	substrateAccount, err := cfg.FindSubstrateAccount(ctx)
	ui.Must(err)
	var substrateAccountId string
	if substrateAccount != nil { // might be nil prior to running `substrate setup`
		substrateAccountId = aws.ToString(substrateAccount.Id)
	}

	// Collate the Substrate-managed roles from all the AWS accounts into
	// compact singular definitions of what they are.
	collated := make(map[string]struct {
		ManagedAssumeRolePolicy  *roles.ManagedAssumeRolePolicy
		ManagedPolicyAttachments *roles.ManagedPolicyAttachments
		Selection                *accounts.Selection
	})
	for _, roleName := range roleNames {
		if _, ok := collated[roleName]; !ok {
			collated[roleName] = struct {
				ManagedAssumeRolePolicy  *roles.ManagedAssumeRolePolicy
				ManagedPolicyAttachments *roles.ManagedPolicyAttachments
				Selection                *accounts.Selection
			}{
				&roles.ManagedAssumeRolePolicy{},
				&roles.ManagedPolicyAttachments{},
				&accounts.Selection{},
			}
		}
		managedAssumeRolePolicy := collated[roleName].ManagedAssumeRolePolicy
		managedPolicyAttachments := collated[roleName].ManagedPolicyAttachments
		selection := collated[roleName].Selection

		for _, tn := range tree[roleName] {
			account := tn.Account
			policyARNs := tn.PolicyARNs
			role := tn.Role

			// Derive the account selection flags from the selectors stored
			// in the SubstrateAccountSelectors tag on the role.
			selectors := strings.Split(role.Tags[tagging.SubstrateAccountSelectors], " ")
			for _, selector := range selectors {
				switch selector {
				case "all-domains":
					selection.AllDomains = true
				case "domain":
					domain := account.Tags[tagging.Domain]
					if naming.Index(selection.Domains, domain) < 0 {
						selection.Domains = append(selection.Domains, domain)
					}
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
					if substrateAccount == nil {
						selection.Admin = true // before `substrate setup`, -substrate won't select anything
					} else {
						selection.Substrate = true // after `substrate setup`, -admin won't select anything
					}
				case "substrate": // not "Substrate" because it's referencing the -substrate flag
					selection.Substrate = true
				case "humans":
					selection.Humans = true
				case "management":
					selection.Management = true
				case "special":
					special := account.Tags[tagging.SubstrateSpecialAccount]
					if naming.Index(selection.Specials, special) < 0 {
						selection.Specials = append(selection.Specials, special)
					}
				case "number":
					number := aws.ToString(account.Id)
					if naming.Index(selection.Numbers, number) < 0 {
						selection.Numbers = append(selection.Numbers, number)
					}
				default:
					ui.Printf("unknown account selector %q", selector)
				}
			}
			ui.Must(selection.Sort())

			// Derive most assume-role policy flags from the statements in the
			// assume-role policy.
			for _, statement := range role.AssumeRolePolicy.Statement {

				// -humans
				var (
					credentialFactory, intranet  bool // must have both of these...
					substrateRole, substrateUser bool // ...or both of these...
					ec2                          bool // ...and this to be detected as -humans
				)
				for _, arn := range statement.Principal.AWS {

					// From before 2023.08 but need to hang around potentially
					// forever so we can detect -humans even on very old roles.
					if strings.HasSuffix(arn, fmt.Sprintf(":user/%s", users.CredentialFactory)) {
						credentialFactory = true
					}
					if strings.HasSuffix(arn, fmt.Sprintf(":role/%s", roles.Intranet)) {
						intranet = true
					}

					if arn == roles.ARN(substrateAccountId, roles.Substrate) {
						substrateRole = true
					}
					if arn == users.ARN(substrateAccountId, users.Substrate) {
						substrateUser = true
					}
				}
				for _, service := range statement.Principal.Service {
					if service == "ec2.amazonaws.com" {
						ec2 = true
					}
				}
				if len(statement.Principal.AWS) == 2 && credentialFactory && intranet && len(statement.Principal.Service) == 1 && ec2 {
					managedAssumeRolePolicy.Humans = true
				} else if len(statement.Principal.AWS) == 2 && substrateRole && substrateUser && len(statement.Principal.Service) == 1 && ec2 {
					managedAssumeRolePolicy.Humans = true
				} else {

					// -aws-service "..."
					// This is nested in the else-statement because the
					// "Service" array is overloaded - used by both -humans
					// and -aws-service - and we need to tell the difference.
					for _, service := range statement.Principal.Service {
						if naming.Index(managedAssumeRolePolicy.AWSServices, service) < 0 {
							managedAssumeRolePolicy.AWSServices = append(managedAssumeRolePolicy.AWSServices, service)
						}
					}

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
								parts := strings.Split(value, ":")
								if len(parts) != 3 || parts[0] != "repo" || parts[2] != "*" {
									continue
								}
								if naming.Index(managedAssumeRolePolicy.GitHubActions, parts[1]) < 0 {
									managedAssumeRolePolicy.GitHubActions = append(managedAssumeRolePolicy.GitHubActions, parts[1])
								}
							}
						}
					}
				}

			}

			// Derive the -assume-role-policy flag from the
			// SubstrateAssumeRolePolicyFilenames tag, if present.
			if filenames, ok := role.Tags[tagging.SubstrateAssumeRolePolicyFilenames]; ok && filenames != "" {
				for _, filename := range strings.Split(filenames, " ") {
					if naming.Index(managedAssumeRolePolicy.Filenames, filename) < 0 {
						managedAssumeRolePolicy.Filenames = append(managedAssumeRolePolicy.Filenames, filename)
					}
				}
			}

			// Derive the policy flags from the policies attached to the role
			// plus the SubstratePolicyAttachmentFilenames tag, if present.
			for _, arn := range policyARNs {
				if arn == "arn:aws:iam::aws:policy/AdministratorAccess" {
					managedPolicyAttachments.AdministratorAccess = true
				} else if arn == "arn:aws:iam::aws:policy/ReadOnlyAccess" {
					managedPolicyAttachments.ReadOnlyAccess = true
				} else if naming.Index(managedPolicyAttachments.ARNs, arn) < 0 {
					managedPolicyAttachments.ARNs = append(managedPolicyAttachments.ARNs, arn)
				}
			}
			if filenames, ok := role.Tags[tagging.SubstratePolicyAttachmentFilenames]; ok && filenames != "" {
				for _, filename := range strings.Split(filenames, " ") {
					if naming.Index(managedPolicyAttachments.Filenames, filename) < 0 {
						managedPolicyAttachments.Filenames = append(managedPolicyAttachments.Filenames, filename)
					}
				}
			}

			managedAssumeRolePolicy.Sort()
			managedPolicyAttachments.Sort()
		}
	}

	switch format.String() {

	case cmdutil.SerializationFormatJSON:
		doc := make([]struct {
			RoleName          string
			AccountSelection  *accounts.Selection
			AssumeRolePolicy  *roles.ManagedAssumeRolePolicy
			PolicyAttachments *roles.ManagedPolicyAttachments
			RoleARNs          []string
		}, len(roleNames))
		for i, roleName := range roleNames {
			doc[i].RoleName = roleName
			doc[i].AccountSelection = collated[roleName].Selection
			doc[i].AssumeRolePolicy = collated[roleName].ManagedAssumeRolePolicy
			doc[i].PolicyAttachments = collated[roleName].ManagedPolicyAttachments
			doc[i].RoleARNs = make([]string, len(tree[roleName]))
			for j, tn := range tree[roleName] {
				doc[i].RoleARNs[j] = tn.Role.ARN
			}
		}
		jsonutil.PrettyPrint(w, doc)

	case cmdutil.SerializationFormatShell:
		fmt.Fprintln(w, "set -e -x")
		for _, roleName := range roleNames {
			fmt.Fprintln(w, strings.Join(
				append(
					append(
						append(
							[]string{fmt.Sprintf("substrate create-role -role %q", roleName)},
							collated[roleName].Selection.Arguments()...,
						),
						collated[roleName].ManagedAssumeRolePolicy.Arguments()...,
					),
					collated[roleName].ManagedPolicyAttachments.Arguments()...,
				),
				" ",
			))
		}

	case cmdutil.SerializationFormatText:
		for i, roleName := range roleNames {
			if i > 0 {
				ui.Print("")
			}
			ui.Print(roleName)
			ui.Print("\taccount selection flags:  ", collated[roleName].Selection)
			ui.Print("\tassume role policy flags: ", collated[roleName].ManagedAssumeRolePolicy)
			ui.Print("\tpolicy attachment flags:  ", collated[roleName].ManagedPolicyAttachments)
			ui.Print("\trole ARNs:")
			for _, tn := range tree[roleName] {
				ui.Print("\t\t", tn.Role.ARN)
			}
		}

	default:
		ui.Fatalf("-format %q not supported", format)
	}
}

type treeNode struct {
	Account    *awsorgs.Account
	PolicyARNs []string
	Role       *awsiam.Role
}
