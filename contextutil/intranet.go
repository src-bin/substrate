package contextutil

import "context"

func IsIntranet(ctx context.Context) bool {
	return ValueString(ctx, Command) == "substrate-intranet"
}
