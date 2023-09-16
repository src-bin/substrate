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
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		StatusCode: http.StatusOK, // ode to Cal Henderson
	}, nil
}

func ErrorResponse2(err error) (*events.APIGatewayV2HTTPResponse, error) {
	body, err := RenderHTML(errorTemplate(), err)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
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
		Headers:    map[string]string{"Content-Type": "application/json; charset=utf-8"},
		StatusCode: statusCode,
	}, nil
}

func ErrorResponseJSON2(statusCode int, err error) (*events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(struct{ Error string }{err.Error()})
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json; charset=utf-8"},
		StatusCode: statusCode,
	}, nil
}
