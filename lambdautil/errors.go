package lambdautil

import (
	"encoding/json"
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
		StatusCode: http.StatusOK, // ode to Cal Henderson
	}, nil
}

func ErrorResponseJSON(statusCode int, err error) (*events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(struct{ Error string }{err.Error()})
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: statusCode,
	}, nil
}
