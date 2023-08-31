package awsiam

import "github.com/src-bin/substrate/awsiam/awsiamusers"

const (
	DeleteConflict      = "DeleteConflict"
	EntityAlreadyExists = awsiamusers.EntityAlreadyExists
	InvalidInput        = "InvalidInput"
	LimitExceeded       = "LimitExceeded"
	NoSuchEntity        = awsiamusers.NoSuchEntity
)
