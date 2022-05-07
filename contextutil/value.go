package contextutil

import "context"

func ValueString(ctx context.Context, key string) string {
	value, _ := ctx.Value(key).(string) // let it be empty if it wants
	return value
}
