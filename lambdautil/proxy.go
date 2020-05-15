package lambdautil

type ProxyEvent struct {
	Body                  string                 `json:"body"`
	Headers               map[string]string      `json:"headers"`
	HTTPMethod            string                 `json:"httpMethod"`
	IsBase64Encoded       bool                   `json:"isBase64Encoded"`
	PathParameters        map[string]string      `json:"pathParameters"`
	Path                  string                 `json:"path"`
	QueryStringParameters map[string]string      `json:"queryStringParameters"`
	RequestContext        map[string]interface{} `json:"requestContext"` // there's structure here but we don't need it (yet)
	Resource              string                 `json:"resource"`
	StageVariables        map[string]string      `json:"stageVariables"`
}

type ProxyResponse struct {
	Body            string            `json:"body"`
	Headers         map[string]string `json:"headers"`
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
}
