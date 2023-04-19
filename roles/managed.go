package roles

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/src-bin/substrate/cmdutil"
)

type ManagedAssumeRolePolicy struct {
	Humans        bool
	AWSServices   []string
	GitHubActions []string
	Filenames     []string
}

func (p *ManagedAssumeRolePolicy) Arguments() []string {
	var ss []string
	if p.Humans {
		ss = append(ss, "-humans")
	}
	for _, service := range p.AWSServices {
		ss = append(ss, "-aws-service", fmt.Sprintf("%q", service))
	}
	for _, githubActions := range p.GitHubActions {
		ss = append(ss, "-github-actions", fmt.Sprintf("%q", githubActions))
	}
	for _, filename := range p.Filenames {
		ss = append(ss, "-assume-role-policy", fmt.Sprintf("%q", filename))
	}
	return ss
}

func (p *ManagedAssumeRolePolicy) GitHubActionsSubs() ([]string, error) {
	subs := make([]string, len(p.GitHubActions))
	for i, repo := range p.GitHubActions {
		if !strings.Contains(repo, "/") {
			return nil, ManagedAssumeRolePolicyError(`-github-actions "..." must contain a '/'`)
		}
		subs[i] = fmt.Sprintf("repo:%s:*", repo)
	}
	return subs, nil
}

func (p *ManagedAssumeRolePolicy) Sort() {
	sort.Strings(p.AWSServices)
	sort.Strings(p.GitHubActions)
	sort.Strings(p.Filenames)
}

func (p *ManagedAssumeRolePolicy) String() string {
	return strings.Join(p.Arguments(), " ")
}

type ManagedAssumeRolePolicyError string

func (err ManagedAssumeRolePolicyError) Error() string {
	return fmt.Sprint("ManagedAssumeRolePolicyError: ", string(err))
}

type ManagedAssumeRolePolicyFlags struct {
	Humans        *bool
	AWSServices   *cmdutil.StringSliceFlag
	GitHubActions *cmdutil.StringSliceFlag
	Filenames     *cmdutil.StringSliceFlag
}

func NewManagedAssumeRolePolicyFlags(u ManagedAssumeRolePolicyFlagsUsage) *ManagedAssumeRolePolicyFlags {
	if u.Humans == "" {
		panic("ManagedAssumeRolePolicyFlagsUsage.Humans can't be empty")
	}
	if u.AWSServices == "" {
		panic("ManagedAssumeRolePolicyFlagsUsage.AWSServices can't be empty")
	}
	if u.GitHubActions == "" {
		panic("ManagedAssumeRolePolicyFlagsUsage.GitHubActions can't be empty")
	}
	if u.Filenames == "" {
		panic("ManagedAssumeRolePolicyFlagsUsage.Filenames can't be empty")
	}
	return &ManagedAssumeRolePolicyFlags{
		Humans:        flag.Bool("humans", false, u.Humans),
		AWSServices:   cmdutil.StringSlice("aws-service", u.AWSServices),
		GitHubActions: cmdutil.StringSlice("github-actions", u.GitHubActions),
		Filenames:     cmdutil.StringSlice("assume-role-policy", u.Filenames),
	}
}

func (f *ManagedAssumeRolePolicyFlags) ManagedAssumeRolePolicy() (*ManagedAssumeRolePolicy, error) {
	if !flag.Parsed() {
		panic("(*ManagedAssumeRolePolicyFlags).ManagedAssumeRolePolicy called before flag.Parse")
	}

	return &ManagedAssumeRolePolicy{
		Humans:        *f.Humans,
		AWSServices:   f.AWSServices.Slice(),
		GitHubActions: f.GitHubActions.Slice(),
		Filenames:     f.Filenames.Slice(),
	}, nil
}

type ManagedAssumeRolePolicyFlagsUsage struct {
	Humans        string
	AWSServices   string
	GitHubActions string
	Filenames     string
}

type ManagedPolicyAttachments struct {
	AdministratorAccess bool
	ReadOnlyAccess      bool
	ARNs                []string
	Filenames           []string
}

func (a *ManagedPolicyAttachments) Arguments() []string {
	var ss []string
	if a.AdministratorAccess {
		ss = append(ss, "-administrator-access")
	}
	if a.ReadOnlyAccess {
		ss = append(ss, "-read-only-access")
	}
	for _, arn := range a.ARNs {
		ss = append(ss, "-policy-arn", fmt.Sprintf("%q", arn))
	}
	for _, filename := range a.Filenames {
		ss = append(ss, "-policy", fmt.Sprintf("%q", filename))
	}
	return ss
}

func (a *ManagedPolicyAttachments) Sort() {
	sort.Strings(a.ARNs)
	sort.Strings(a.Filenames)
}

func (a *ManagedPolicyAttachments) String() string {
	return strings.Join(a.Arguments(), " ")
}

type ManagedPolicyAttachmentsError string

func (err ManagedPolicyAttachmentsError) Error() string {
	return fmt.Sprint("ManagedPolicyAttachmentsError: ", string(err))
}

type ManagedPolicyAttachmentsFlags struct {
	AdministratorAccess *bool
	ReadOnlyAccess      *bool
	ARNs                *cmdutil.StringSliceFlag
	Filenames           *cmdutil.StringSliceFlag
}

func NewManagedPolicyAttachmentsFlags(u ManagedPolicyAttachmentsFlagsUsage) *ManagedPolicyAttachmentsFlags {
	if u.AdministratorAccess == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.AdministratorAccess can't be empty")
	}
	if u.ReadOnlyAccess == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.ReadOnlyAccess can't be empty")
	}
	if u.ARNs == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.ARNs can't be empty")
	}
	if u.Filenames == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.Filenames can't be empty")
	}
	return &ManagedPolicyAttachmentsFlags{
		AdministratorAccess: flag.Bool("administrator-access", false, u.AdministratorAccess),
		ReadOnlyAccess:      flag.Bool("read-only-access", false, u.ReadOnlyAccess),
		ARNs:                cmdutil.StringSlice("policy-arn", u.ARNs),
		Filenames:           cmdutil.StringSlice("policy", u.Filenames),
	}
}

func (f *ManagedPolicyAttachmentsFlags) ManagedPolicyAttachments() (*ManagedPolicyAttachments, error) {
	if !flag.Parsed() {
		panic("(*ManagedPolicyAttachmentsFlags).ManagedPolicyAttachments called before flag.Parse")
	}

	if *f.AdministratorAccess && *f.ReadOnlyAccess {
		return nil, ManagedPolicyAttachmentsError("can't provide both -administrator and -read-only")
	}

	return &ManagedPolicyAttachments{
		AdministratorAccess: *f.AdministratorAccess,
		ReadOnlyAccess:      *f.ReadOnlyAccess,
		ARNs:                f.ARNs.Slice(),
		Filenames:           f.Filenames.Slice(),
	}, nil
}

type ManagedPolicyAttachmentsFlagsUsage struct {
	AdministratorAccess string
	ReadOnlyAccess      string
	ARNs                string
	Filenames           string
}
