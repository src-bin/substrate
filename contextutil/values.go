package contextutil

import "context"

const (
	Command    = "Command"
	Subcommand = "Subcommand"
	Username   = "Username"

	RedirectStdoutTo = "RedirectStdoutTo"
)

func WithValues(ctx context.Context, command, subcommand, username string) context.Context {
	return context.WithValue(
		context.WithValue(
			context.WithValue(
				ctx,
				Command,
				command,
			),
			Subcommand,
			subcommand,
		),
		Username,
		username,
	)
}

func ValueString(ctx context.Context, key string) string {
	value, _ := ctx.Value(key).(string) // let it be empty if it wants
	return value
}
