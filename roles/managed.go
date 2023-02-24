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
		ss = append(ss, "-aws-service", service)
	}
	for _, githubActions := range p.GitHubActions {
		ss = append(ss, "-github-actions", githubActions)
	}
	for _, filename := range p.Filenames {
		ss = append(ss, "-assume-role-policy", filename)
	}
	return ss
}

func (p *ManagedAssumeRolePolicy) Sort() {
	sort.Strings(p.AWSServices)
	sort.Strings(p.GitHubActions)
	sort.Strings(p.Filenames)
}

func (p *ManagedAssumeRolePolicy) String() string {
	return strings.Join(p.Arguments(), " ")
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
	FullAccess bool
	ReadOnly   bool
	ARNs       []string
	Filenames  []string
}

func (a *ManagedPolicyAttachments) Arguments() []string {
	var ss []string
	if a.FullAccess {
		ss = append(ss, "-full-access")
	}
	if a.ReadOnly {
		ss = append(ss, "-read-only")
	}
	for _, arn := range a.ARNs {
		ss = append(ss, "-policy-arn", arn)
	}
	for _, filename := range a.Filenames {
		ss = append(ss, "-policy", filename)
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
	FullAccess *bool
	ReadOnly   *bool
	ARNs       *cmdutil.StringSliceFlag
	Filenames  *cmdutil.StringSliceFlag
}

func NewManagedPolicyAttachmentsFlags(u ManagedPolicyAttachmentsFlagsUsage) *ManagedPolicyAttachmentsFlags {
	if u.FullAccess == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.FullAccess can't be empty")
	}
	if u.ReadOnly == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.ReadOnly can't be empty")
	}
	if u.ARNs == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.ARNs can't be empty")
	}
	if u.Filenames == "" {
		panic("ManagedPolicyAttachmentsFlagsUsage.Filenames can't be empty")
	}
	return &ManagedPolicyAttachmentsFlags{
		FullAccess: flag.Bool("full-access", false, u.FullAccess),
		ReadOnly:   flag.Bool("read-only", false, u.ReadOnly),
		ARNs:       cmdutil.StringSlice("policy-arn", u.ARNs),
		Filenames:  cmdutil.StringSlice("policy", u.Filenames),
	}
}

func (f *ManagedPolicyAttachmentsFlags) ManagedPolicyAttachments() (*ManagedPolicyAttachments, error) {
	if !flag.Parsed() {
		panic("(*ManagedPolicyAttachmentsFlags).ManagedPolicyAttachments called before flag.Parse")
	}

	if *f.FullAccess && *f.ReadOnly {
		return nil, ManagedPolicyAttachmentsError("can't provide both -full-access and -read-only")
	}

	return &ManagedPolicyAttachments{
		FullAccess: *f.FullAccess,
		ReadOnly:   *f.ReadOnly,
		ARNs:       f.ARNs.Slice(),
		Filenames:  f.Filenames.Slice(),
	}, nil
}

type ManagedPolicyAttachmentsFlagsUsage struct {
	FullAccess string
	ReadOnly   string
	ARNs       string
	Filenames  string
}
