package awscfg

import "github.com/aws/aws-sdk-go-v2/config"

type Config struct {
	cfg config.EnvConfig // TODO facet by account
	// TODO svcs, too
}
