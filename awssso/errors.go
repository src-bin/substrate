package awssso

import "fmt"

const (
	AccessDeniedException = "AccessDeniedException"
	ConflictException     = "ConflictException"
)

type NotFound [2]string // type, identifier

func (err NotFound) Error() string {
	return fmt.Sprintf("%s %s not found", err[0], err[1])
}
