package authorizerutil

// event.RequestContext.Authorizer keys.
const (
	AccessToken = "AccessToken"
	IDToken     = "IDToken"
	PrincipalId = "principalId" // lowercase because that's how it was in API Gateway v1
	RoleName    = "RoleName"

	Error = "Error"

	Location = "Location"
)
