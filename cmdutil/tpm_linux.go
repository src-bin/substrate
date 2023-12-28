//go:build linux
// +build linux

package cmdutil

import "github.com/aws/aws-sdk-go-v2/aws"

// Linux users have to suck it up and set environment variables.
func SetTPM(aws.Credentials) error { return nil }
func SetenvFromTPM(string) error   { return nil }
