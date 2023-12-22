package cmdutil

import (
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/ui"
)

func DomainFlag(usage string) (
	*string,
	*pflag.Flag,
	func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective),
) {
	f := &domainFlag{}
	return &f.domain, &pflag.Flag{
		Name:      "domain",
		Shorthand: "d",
		Usage:     usage,
		Value:     f,
	}, f.CompletionFunc
}

func EnvironmentFlag(usage string) (
	*string,
	*pflag.Flag,
	func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective),
) {
	f := &environmentFlag{}
	return &f.environment, &pflag.Flag{
		Name:      "environment",
		Shorthand: "e",
		Usage:     usage,
		Value:     f,
	}, f.CompletionFunc
}

type Format string

const (
	FormatEnv               Format = "env"
	FormatExport            Format = "export"
	FormatExportWithHistory Format = "export-with-history"
	FormatJSON              Format = "json"
	FormatShell             Format = "shell"
	FormatText              Format = "text"
)

func FormatFlag(defaultFormat Format, validFormats []Format) (
	*Format,
	*pflag.Flag,
	func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective),
) {
	var ss []string
	for _, v := range validFormats {
		switch v {
		case FormatExport:
			ss = append(ss, "export (exported shell environment variables)")
		case FormatExportWithHistory:
			ss = append(ss, "export-with-history (exported shell environment variables and an unassume-role alias to go back)")
		case FormatEnv:
			ss = append(ss, "env (.env file)")
		case FormatJSON:
			ss = append(ss, "json")
		case FormatShell:
			ss = append(ss, "shell (executable shell commands)")
		case FormatText:
			ss = append(ss, "text (for human-readable plaintext)")
		}
	}
	f := &formatFlag{defaultFormat, validFormats}
	return &f.format, &pflag.Flag{
		Name:     "format",
		Usage:    fmt.Sprint("output format - ", strings.Join(ss, ", ")),
		DefValue: string(defaultFormat),
		Value:    f,
	}, f.CompletionFunc
}

type FormatFlagError Format

func (err FormatFlagError) Error() string {
	return fmt.Sprintf("--format %q not supported", string(err))
}

func NoCompletionFunc(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

func QualityFlag(usage string) (
	*string,
	*pflag.Flag,
	func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective),
) {
	f := &qualityFlag{}
	if qualities, _ := naming.Qualities(); len(qualities) == 1 {
		f.Set(qualities[0])
	}
	return &f.quality, &pflag.Flag{
		Name: "quality",
		// Shorthand: "q", // taken by --quiet's shorthand which is less confusing and more idiomatic
		Usage: usage,
		Value: f,
	}, f.CompletionFunc
}

func QuietFlag() *pflag.Flag {
	return &pflag.Flag{
		Name:        "quiet",
		Shorthand:   "q",
		Usage:       "suppress status and diagnostic output",
		Value:       &quietFlag{},
		DefValue:    "false",
		NoOptDefVal: "true",
	}
}

// TODO remove in favor of pflag.StringArray or pflag.StringSlice; I don't understand the difference yet.
type StringSliceFlag []string

func StringSlice(name, usage string) *StringSliceFlag {
	ss := &StringSliceFlag{}
	flag.Var(ss, name, usage)
	return ss
}
func (ssf *StringSliceFlag) Len() int {
	if ssf == nil || *ssf == nil {
		return 0
	}
	return len(*ssf)
}
func (ssf *StringSliceFlag) Set(s string) error {
	*ssf = append(*ssf, s)
	return nil
}
func (ssf *StringSliceFlag) Slice() []string {
	if ssf == nil || *ssf == nil {
		return []string{}
	}
	return append([]string{}, *ssf...)
}
func (ssf *StringSliceFlag) String() string {
	return strings.Join(*ssf, ", ")
}

type domainFlag struct {
	domain string
}

func (d *domainFlag) CompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{}, cobra.ShellCompDirectiveNoFileComp // TODO shell completion for domains
}

func (d *domainFlag) Set(domain string) error {
	d.domain = domain
	return nil
}

func (d *domainFlag) String() string {
	return d.domain
}

func (*domainFlag) Type() string {
	return "<domain>"
}

type environmentFlag struct {
	environment string
}

func (e *environmentFlag) CompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	environments, err := naming.Environments()
	ui.Must(err)
	return environments, cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoFileComp
}

func (e *environmentFlag) Set(environment string) error {
	e.environment = environment
	return nil
}

func (e *environmentFlag) String() string {
	return e.environment
}

func (*environmentFlag) Type() string {
	return "<environment>"
}

type formatFlag struct {
	format       Format
	validFormats []Format
}

func (f *formatFlag) CompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ss := make([]string, len(f.validFormats))
	for i, format := range f.validFormats {
		ss[i] = string(format)
	}
	return ss, cobra.ShellCompDirectiveNoFileComp
}

func (f *formatFlag) Set(s string) error {
	format := Format(s)
	for _, v := range f.validFormats {
		if format == v {
			f.format = format
			return nil
		}
	}
	return FormatFlagError(format)
}

func (f *formatFlag) String() string {
	if f.format == "" {
		return string(FormatExport)
	}
	return string(f.format)
}

func (*formatFlag) Type() string {
	return "<format>"
}

type qualityFlag struct {
	quality string
}

func (q *qualityFlag) CompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	qualities, err := naming.Qualities()
	ui.Must(err)
	return qualities, cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoFileComp
}

func (q *qualityFlag) Set(quality string) error {
	q.quality = quality
	return nil
}

func (q *qualityFlag) String() string {
	return q.quality
}

func (*qualityFlag) Type() string {
	return "<quality>"
}

type quietFlag struct {
	quiet bool
}

func (q *quietFlag) Set(s string) error {
	old := q.quiet
	if q.quiet = s == "true"; q.quiet {
		ui.Quiet()
	} else if old {
		return fmt.Errorf("can't turn off quiet mode")
	}
	return nil
}

func (q *quietFlag) String() string {
	if q.quiet {
		return "true"
	}
	return "false"
}

func (*quietFlag) Type() string {
	return "bool"
}
