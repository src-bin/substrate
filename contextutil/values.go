package contextutil

import "context"

const (
	Command    = "Comand"
	Subcommand = "Subcommand"
	Username   = "Username"
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
