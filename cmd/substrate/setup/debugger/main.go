package debugger

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

var (
	console = new(bool)
	shell   = new(string)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debugger [--console] [--shell <shell>]",
		Short: "TODO debugger.Command().Short",
		Long:  `TODO debugger.Command().Long`,
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--console",
				"--shell",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().BoolVar(console, "console", false, "open the AWS Console instead of executing a shell")
	cmd.Flags().StringVar(shell, "shell", "", "pathname of the shell to run instead of SHELL from the environment")
	cmd.RegisterFlagCompletionFunc("shell", cmdutil.NoCompletionFunc)
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {
	if *shell == "" {
		*shell = os.Getenv("SHELL")
	}
	if *shell == "" {
		ui.Print("SHELL wasn't in the environment; launching /bin/sh; override by setting SHELL or passing --shell")
		*shell = "/bin/sh"
	}

	if callerIdentity, err := cfg.GetCallerIdentity(ctx); err == nil {
		ui.Printf("initial identity: %s", callerIdentity.Arn)
	}
	regions.Default()
	ui.Must2(cfg.BootstrapCredentials(ctx)) // get from anywhere to IAM credentials so we can assume roles
	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.Substrate, // triggers affordances for using (deprecated) OrganizationAdministrator role, too
		time.Hour,
	))
	ui.Printf("management identity: %s", mgmtCfg.MustGetCallerIdentity(ctx).Arn)

	substrateCfg, err := mgmtCfg.AssumeSubstrateRole(ctx, roles.Substrate, time.Hour)
	if err != nil {
		substrateCfg, err = mgmtCfg.AssumeSubstrateRole(ctx, roles.Administrator, time.Hour)
	}
	if err != nil {
		substrateCfg, err = mgmtCfg.AssumeSubstrateRole(ctx, roles.OrganizationAccountAccessRole, time.Hour)
	}

	var creds aws.Credentials
	if err == nil {
		ui.Printf("Substrate identity: %s", substrateCfg.MustGetCallerIdentity(ctx).Arn)
		ui.Printf("running %s with AWS credentials from your Substrate account in the environment", *shell)
		creds, err = substrateCfg.Retrieve(ctx)
	} else {
		ui.Printf("running %s with AWS credentials from your management account in the environment", *shell)
		creds, err = mgmtCfg.Retrieve(ctx)
	}
	ui.Must(err)

	if *console {
		consoleSigninURL, err := federation.ConsoleSigninURL(
			creds,
			"", // destination (empty means the AWS Console homepage)
			nil,
		)
		if err != nil {
			ui.Fatal(err)
		}
		ui.OpenURL(consoleSigninURL)
		return
	}

	cmd := &exec.Cmd{
		Env: append(
			os.Environ(),
			fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID),
			fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey),
			fmt.Sprintf("AWS_SESSION_TOKEN=%s", creds.SessionToken),
			fmt.Sprintf("SHELL=%s", *shell),
		),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	cmd.Path, err = exec.LookPath(*shell)
	ui.Must(err)
	cmd.Args = []string{fmt.Sprintf("-%s", filepath.Base(cmd.Path))}
	ui.Must(cmd.Run())

}
