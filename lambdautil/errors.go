package lambdautil

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

//go:generate go run ../tools/template/main.go -name errorTemplate error.html

func ErrorResponse(err error) (*events.APIGatewayProxyResponse, error) {
	body, err := RenderHTML(errorTemplate(), err)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil
}
