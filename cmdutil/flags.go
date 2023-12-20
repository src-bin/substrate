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

func DomainFlag(usage string) *string {
	return flag.String("domain", "", usage)
}

func EnvironmentFlag(usage string) *string {
	return flag.String("environment", "", usage)
}

const (
	FormatEnv               = "env"
	FormatExport            = "export"
	FormatExportWithHistory = "export-with-history" // undocumented because it only really makes sense as used for credentials by substrate assume-role
	FormatJSON              = "json"
	FormatShell             = "shell"
	FormatText              = "text" // undocumented default for some tools
)

func FormatFlag(defaultFormat string, validFormats []string) *formatFlag {
	return &formatFlag{defaultFormat, validFormats}
}

type FormatFlagError string

func (err FormatFlagError) Error() string {
	return fmt.Sprintf("--format %q not supported", string(err))
}

func QualityFlag(usage string) *string {
	quality := ""
	if qualities, _ := naming.Qualities(); len(qualities) == 1 {
		quality = qualities[0]
	}
	return flag.String("quality", quality, usage)
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

type formatFlag struct {
	format       string
	validFormats []string
}

func (f *formatFlag) CompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return f.validFormats, cobra.ShellCompDirectiveNoFileComp
}

func (f *formatFlag) Set(format string) error {
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
		return FormatExport
	}
	return f.format
}

func (*formatFlag) Type() string {
	return "<format>"
}

func (f *formatFlag) Usage() string {
	var ss []string
	for _, v := range f.validFormats {
		switch v {
		case FormatExport:
			ss = append(ss, "export (exported shell environment variables)")
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
	return fmt.Sprint("output format - ", strings.Join(ss, ", "))
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
