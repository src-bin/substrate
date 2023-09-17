package cmdutil

import (
	"flag"
	"fmt"
)

const (
	SerializationFormatEnv               = "env"
	SerializationFormatExport            = "export"
	SerializationFormatExportWithHistory = "export-with-history" // undocumented because it only really makes sense as used for credentials by substrate-assume-role
	SerializationFormatJSON              = "json"
	SerializationFormatShell             = "shell"
	SerializationFormatText              = "text" // undocumented default for some tools
)

type SerializationFormat struct {
	format string
}

func SerializationFormatFlag(defaultFormat, usage string) *SerializationFormat {
	f := &SerializationFormat{defaultFormat}
	flag.Var(f, "format", usage)
	return f
}

func (f *SerializationFormat) Set(format string) error {
	switch format {
	case SerializationFormatEnv, SerializationFormatExport, SerializationFormatExportWithHistory, SerializationFormatJSON, SerializationFormatShell, SerializationFormatText:
	default:
		return SerializationFormatError(format)
	}
	f.format = format
	return nil
}

func (f *SerializationFormat) String() string {
	if f.format == "" {
		return SerializationFormatExport
	}
	return f.format
}

type SerializationFormatError string

func (err SerializationFormatError) Error() string {
	return fmt.Sprintf(`-format %q not supported`, string(err))
}

const SerializationFormatUsage = `output format - "export" for exported shell environment variables, "env" for .env files, "json" for JSON, "shell" for shell commands, and "text" for human-readable plaintext`
