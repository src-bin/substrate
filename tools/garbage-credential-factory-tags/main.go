// garbage-credential-factory-tags fills up the 50 available tags on the
// CredentialFactory user with garbage to exercise the tag garbage collector
// that runs in the Intranet's /credential-factory/authorize endpoint. Run
// it like this:
//
// 1. `eval $(substrate credentials)` in your Substrate repository
// 2. `cd ../substrate` (or wherever you put it)
// 3. `go run tools/garbage-credential-factory-tags/main.go`
//
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/randutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	TagKeyPrefix   = "CredentialFactory:"
	TagValueFormat = "%s %s expiry %s"
)

func main() {
	ctx := context.Background()
	cfg, err := awscfg.NewConfig(ctx)
	if err != nil {
		ui.Fatal(err)
	}
	cfg = cfg.Regional("us-west-2") // XXX good for testing in src-bin-test1

	for i := 0; i < 47; i++ {
		if err = awsiam.TagUser(
			ctx,
			cfg,
			users.CredentialFactory,
			TagKeyPrefix+randutil.String(),
			fmt.Sprintf(
				TagValueFormat,
				"example@example.com",
				"Nobody",
				time.Now().Add(-time.Hour).Format(time.RFC3339),
			),
		); err != nil {
			break
		}
		fmt.Fprint(os.Stderr, ".")
	}
	fmt.Fprintln(os.Stderr, "")
	if err != nil {
		ui.Fatal(err)
	}
}
