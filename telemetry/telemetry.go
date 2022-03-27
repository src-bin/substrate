package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const Filename = "substrate.telemetry"

// Endpoint is an HTTP(S) URL where telemetry is sent, if it's not an empty
// string. The following values are useful:
// - "": Do not send telemetry.
// - "https://src-bin.org/telemetry/": Send telemetry to staging.
// - "https://src-bin.com/telemetry/": Send telemetry to production.
//
// The actual value is set at build time.
var Endpoint = ""

type Event struct {
	Command, Subcommand                      string // e.g. "substrate" and "assume-role" or "substrate-intranet" and "InstanceFactory"
	Version                                  string
	InitialAccountNumber, FinalAccountNumber string // avoid disclosing domain, environment, and quality
	EmailDomainName                          string // avoid PII in local portion
	InitialRoleName, FinalRoleName           string // "Administrator", "Auditor", or "Other" (avoid disclosing custom role names)
	IsEC2                                    bool
	Format                                   string        `json:",omitempty"` // -format, if applicable
	post, wait                               chan struct{} // `json:"-"`
}

func NewEvent(ctx context.Context) (*Event, error) {
	e := &Event{
		Command:    stringFromContext(ctx, "Command"),
		Subcommand: stringFromContext(ctx, "Subcommand"),
		Version:    version.Version,
		//Format // TODO when cmdutil.SerializationFormat.Set is called
		post: make(chan struct{}),
		wait: make(chan struct{}),
	}

	ctx, _ = context.WithTimeout(ctx, 100*time.Millisecond)
	for _, url := range []string{
		"http://169.254.169.254/latest/api/token",
		"http://[fd00:ec2::254]/latest/api/token",
	} {
		req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
		if err != nil {
			return nil, err
		}
		if _, err := http.DefaultClient.Do(req); err == nil {
			e.IsEC2 = true
			break
		}
	}

	return e, nil
}

func (e *Event) Post(ctx context.Context) error {
	if e == nil || Endpoint == "" {
		return nil
	}
	select {
	case <-e.post:
		return nil
	case <-e.wait:
		return nil
	default:
	}
	close(e.post)

	pathname, err := fileutil.PathnameInParents(Filename)
	if err != nil {
		return nil // surpress this error and just don't post telemetry
	}
	ok, err := ui.ConfirmFile(
		pathname,
		"can Substrate post non-sensitive and non-personally identifying telemetry (documented in more detail at <https://src.bin.com/substrate/manual/telemetry/>) to Source & Binary to better understand how Substrate is being used? (yes/no)",
	)
	if err != nil {
		return err
	}
	if !ok {
		close(e.wait)
		return nil
	}

	b := &bytes.Buffer{}
	if err := json.NewEncoder(b).Encode(e); err != nil {
		return err
	}
	ctx, _ = context.WithTimeout(ctx, time.Second)
	req, err := http.NewRequestWithContext(ctx, "POST", Endpoint, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)
	close(e.wait)

	return err
}

func (e *Event) SetInitialAccountNumber(accountNumber string) {
	e.InitialAccountNumber = accountNumber
	if e.FinalAccountNumber == "" {
		e.FinalAccountNumber = accountNumber
	}
}
func (e *Event) SetFinalAccountNumber(accountNumber string) {
	e.FinalAccountNumber = accountNumber
}

func (e *Event) SetEmailDomainName(email string) {
	if ss := strings.Split(email, "@"); len(ss) == 2 {
		e.EmailDomainName = ss[1]
	}
}

func (e *Event) SetInitialRoleName(roleArn string) (err error) {
	e.InitialRoleName, err = roleNameFromArn(roleArn)
	if e.FinalRoleName == "" {
		e.FinalRoleName, err = roleNameFromArn(roleArn)
	}
	return
}

func (e *Event) SetFinalRoleName(roleArn string) (err error) {
	e.FinalRoleName, err = roleNameFromArn(roleArn)
	return
}

func (e *Event) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.wait:
	}
	return nil
}

func roleNameFromArn(roleArn string) (string, error) {
	a, err := arn.Parse(roleArn)
	if err != nil {
		return "", err
	}
	ss := strings.Split(a.Resource, "/")
	if len(ss) < 2 {
		return "", errors.New("arn: not enough sections") // <https://github.com/aws/aws-sdk-go-v2/blob/v1.15.0/aws/arn/arn.go#L23>
	}
	switch ss[0] {
	case "assumed-role", "role":
	default:
		return "", fmt.Errorf("%q is not an IAM role ARN", roleArn)
	}
	switch ss[1] {
	case roles.Administrator, roles.Auditor:
		return ss[1], nil
	}
	return "Other", nil // don't disclose customer-defined role names
}

func stringFromContext(ctx context.Context, key string) string {
	value, _ := ctx.Value(key).(string) // let it be empty if it wants
	return value
}
