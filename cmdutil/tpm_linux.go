//go:build linux
// +build linux

package cmdutil

// Linux users have to suck it up and set environment variables.
func SetTPM() error        { return nil }
func SetenvFromTPM() error { return nil }
