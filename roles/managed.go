package roles

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/pflag"
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
		ss = append(ss, "--humans")
	}
	for _, service := range p.AWSServices {
		ss = append(ss, "--aws-service", fmt.Sprintf("%q", service))
	}
	for _, githubActions := range p.GitHubActions {
		ss = append(ss, "--github-actions", fmt.Sprintf("%q", githubActions))
	}
	for _, filename := range p.Filenames {
		ss = append(ss, "--assume-role-policy", fmt.Sprintf("%q", filename))
	}
	return ss
}

func (p *ManagedAssumeRolePolicy) FlagSet(u ManagedAssumeRolePolicyFlagsUsage) *pflag.FlagSet {
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
	p.Reset()
	set := pflag.NewFlagSet("[assume-role policy flags]", pflag.ExitOnError)
	set.BoolVar(&p.Humans, "humans", false, u.Humans)
	set.StringArrayVar(&p.AWSServices, "aws-service", []string{}, u.AWSServices)
	set.StringArrayVar(&p.GitHubActions, "github-actions", []string{}, u.GitHubActions)
	set.StringArrayVar(&p.Filenames, "assume-role-policy", []string{}, u.Filenames)
	return set
}

func (p *ManagedAssumeRolePolicy) GitHubActionsSubs() ([]string, error) {
	subs := make([]string, len(p.GitHubActions))
	for i, repo := range p.GitHubActions {
		if !strings.Contains(repo, "/") {
			return nil, ManagedAssumeRolePolicyError(`--github-actions "..." must contain a '/'`)
		}
		subs[i] = fmt.Sprintf("repo:%s:*", repo)
	}
	return subs, nil
}

func (p *ManagedAssumeRolePolicy) Reset() {
	p.Humans = false
	p.AWSServices = []string{}
	p.GitHubActions = []string{}
	p.Filenames = []string{}
}

func (p *ManagedAssumeRolePolicy) Sort() {
	sort.Strings(p.AWSServices)
	sort.Strings(p.GitHubActions)
	sort.Strings(p.Filenames)
}

func (p *ManagedAssumeRolePolicy) String() string {
	return strings.Join(p.Arguments(), " ")
}

func (p *ManagedAssumeRolePolicy) Validate() error {
	return nil
}

type ManagedAssumeRolePolicyError string

func (err ManagedAssumeRolePolicyError) Error() string {
	return fmt.Sprint("ManagedAssumeRolePolicyError: ", string(err))
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
		ss = append(ss, "--administrator-access")
	}
	if a.ReadOnlyAccess {
		ss = append(ss, "--read-only-access")
	}
	for _, arn := range a.ARNs {
		ss = append(ss, "--policy-arn", fmt.Sprintf("%q", arn))
	}
	for _, filename := range a.Filenames {
		ss = append(ss, "--policy", fmt.Sprintf("%q", filename))
	}
	return ss
}

func (a *ManagedPolicyAttachments) FlagSet(u ManagedPolicyAttachmentsFlagsUsage) *pflag.FlagSet {
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
	a.Reset()
	set := pflag.NewFlagSet("[policy attachment flags]", pflag.ExitOnError)
	set.BoolVar(&a.AdministratorAccess, "administrator-access", false, u.AdministratorAccess)
	set.BoolVar(&a.ReadOnlyAccess, "read-only-access", false, u.ReadOnlyAccess)
	set.StringArrayVar(&a.ARNs, "policy-arn", []string{}, u.ARNs)
	set.StringArrayVar(&a.Filenames, "policy", []string{}, u.Filenames)
	return set
}

func (a *ManagedPolicyAttachments) Reset() {
	a.AdministratorAccess = false
	a.ReadOnlyAccess = false
	a.ARNs = []string{}
	a.Filenames = []string{}
}

func (a *ManagedPolicyAttachments) Sort() {
	sort.Strings(a.ARNs)
	sort.Strings(a.Filenames)
}

func (a *ManagedPolicyAttachments) String() string {
	return strings.Join(a.Arguments(), " ")
}

func (a *ManagedPolicyAttachments) Validate() error {
	if a.AdministratorAccess && a.ReadOnlyAccess {
		return ManagedPolicyAttachmentsError("can't provide both --administrator-access and --read-only-access")
	}
	return nil
}

type ManagedPolicyAttachmentsError string

func (err ManagedPolicyAttachmentsError) Error() string {
	return fmt.Sprint("ManagedPolicyAttachmentsError: ", string(err))
}

type ManagedPolicyAttachmentsFlagsUsage struct {
	AdministratorAccess, ReadOnlyAccess, ARNs, Filenames string
}
