package terraform

//go:generate go run ../tools/template/main.go -name intranetGlobalTemplate -o intranet-global.go ../intranet/global
func IntranetGlobalModule() *Directory {
	return &Directory{intranetGlobalTemplate()}
}

//go:generate go run ../tools/template/main.go -name intranetRegionalTemplate -o intranet-regional.go ../intranet/regional
func IntranetRegionalModule() *Directory {
	return &Directory{intranetRegionalTemplate()}
}

//go:generate go run ../tools/template/main.go -name lambdaFunctionGlobalTemplate -o lambda-function-global.go ../lambda-function/global
func LambdaFunctionGlobalModule() *Directory {
	return &Directory{lambdaFunctionGlobalTemplate()}
}

//go:generate go run ../tools/template/main.go -name lambdaFunctionRegionalTemplate -o lambda-function-regional.go ../lambda-function/regional
func LambdaFunctionRegionalModule() *Directory {
	return &Directory{lambdaFunctionRegionalTemplate()}
}