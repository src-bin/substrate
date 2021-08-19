package cmdutil

import (
	"flag"
)

const (
	SerializationFormatEnv               = "env"
	SerializationFormatExport            = "export"
	SerializationFormatExportWithHistory = "export-with-history" // undocumented because it only really makes sense as used for credentials by substrate-assume-role
	SerializationFormatJSON              = "json"
	SerializationFormatText              = "txt" // undocumented default for substrate-whoami
)

type SerializationFormat struct {
	format string
}

func SerializationFormatFlag(format string) *SerializationFormat {
	f := &SerializationFormat{format}
	flag.Var(
		f,
		"format",
		`output format - "export" for exported shell environment variables, "env" for .env files, "json" for JSON`,
	)
	return f
}

func (f *SerializationFormat) Set(format string) error {
	switch format {
	case SerializationFormatEnv, SerializationFormatExport, SerializationFormatExportWithHistory, SerializationFormatJSON:
	default:
		return SerializationFormatError(`supported formats are "export", "env", and "json"`) // and "export-with-history" and "text" but we'll keep those quiet
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
	return string(err)
}
