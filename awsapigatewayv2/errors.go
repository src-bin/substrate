package awsapigatewayv2

import "fmt"

const (
	BadRequestException = "BadRequestException"
	ConflictException   = "ConflictException"
)

type NotFound struct {
	Name, Type string
}

func (err NotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", err.Type, err.Name)
}
