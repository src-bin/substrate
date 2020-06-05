package terraform

//go:generate go run ../tools/template/main.go -name intranetTemplate ../intranet
func IntranetModule() *Directory {
	return &Directory{intranetTemplate()}
}

//go:generate go run ../tools/template/main.go -name lambdaFunctionTemplate ../lambda-function
func LambdaFunctionModule() *Directory {
	return &Directory{lambdaFunctionTemplate()}
}
