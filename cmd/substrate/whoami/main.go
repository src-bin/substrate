package whoami

import (
	"context"
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
)

func Main(ctx context.Context, cfg *awscfg.Main) {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatText) // default to undocumented special value
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	flag.Usage = func() {
		ui.Print("Usage: substrate whoami [-format <format>] [-quiet]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *quiet {
		ui.Quiet()
	}

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	callerIdentity, err := awssts.GetCallerIdentity(sts.New(awssessions.Must(awssessions.NewSession(awssessions.Config{}))))
	if err != nil {
		ui.Fatal(err)
	}
	//log.Printf("%+v", callerIdentity)

	sess, err := awssessions.InManagementAccount(roles.OrganizationReader, awssessions.Config{})
	if err != nil {
		ui.Fatal(err)
	}
	account, err := awsorgs.DescribeAccount(organizations.New(sess), aws.StringValue(callerIdentity.Account))
	if err != nil {
		ui.Fatal(err)
	}
	//log.Printf("%+v", account)

	switch format.String() {
	case cmdutil.SerializationFormatEnv:
		fmt.Printf(
			"DOMAIN=%q\nENVIRONMENT=%q\nQUALITY=%q\nROLE=%q\n",
			account.Tags[tags.Domain],
			account.Tags[tags.Environment],
			account.Tags[tags.Quality],
			aws.StringValue(callerIdentity.Arn),
		)
	case cmdutil.SerializationFormatExport, cmdutil.SerializationFormatExportWithHistory:
		fmt.Printf(
			"export DOMAIN=%q ENVIRONMENT=%q QUALITY=%q ROLE=%q\n",
			account.Tags[tags.Domain],
			account.Tags[tags.Environment],
			account.Tags[tags.Quality],
			aws.StringValue(callerIdentity.Arn),
		)
	case cmdutil.SerializationFormatJSON:
		ui.PrettyPrintJSON(map[string]string{
			tags.Domain:      account.Tags[tags.Domain],
			tags.Environment: account.Tags[tags.Environment],
			tags.Quality:     account.Tags[tags.Quality],
			"Role":           aws.StringValue(callerIdentity.Arn),
		})
	case cmdutil.SerializationFormatText:
		ui.Printf(
			"you're %s in\nDomain:      %s\nEnvironment: %s\nQuality:     %s",
			aws.StringValue(callerIdentity.Arn),
			account.Tags[tags.Domain],
			account.Tags[tags.Environment],
			account.Tags[tags.Quality],
		)
	default:
		ui.Fatalf("-format=%q not supported", format)
	}

}
