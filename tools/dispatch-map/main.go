package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
)

// includeInDispatchMap is the name of the type in cmd/substrate that, when
// declared as the only argument to a package-level function, signals this
// program to include that function in the dispatch map in cmd/substrate.
const includeInDispatchMap = "includeInDispatchMap"

func main() {
	name := flag.String("name", "dispatchMap", "function (or, with -receiver-type, method) name which will return this template")
	out := flag.String("o", "dispatch-map.go", "filename where generated Go code will be written (defaults to \"dispatch-map.go\")")
	pkg := flag.String("package", "main", "package name for the generated Go code (defaults to \"main\")")
	flag.Parse()
	if flag.NArg() > 0 {
		log.Fatal("too many arguments")
	}

	// Find all the package-level functions that take a single argument of the
	// includeInDispatchMap type in the current working directory.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatal("unexpectedly found more than one package in a single directory")
	}
	var dispatchMapKeys []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for name, object := range file.Scope.Objects {
				if object.Kind == ast.Fun {
					if list := object.Decl.(*ast.FuncDecl).Type.Params.List; len(list) == 1 && fmt.Sprint(list[0].Type) == includeInDispatchMap {
						dispatchMapKeys = append(dispatchMapKeys, name)
					}
				}
			}
		}
	}

	// Generate and format Go code that declares the dispatch map. Rewrite
	// camelCase function names in dash-case to match command-line arguments.
	b := &bytes.Buffer{}
	re := regexp.MustCompile(`[\p{Lu}]`)
	fmt.Fprintf(b, "package %s\nvar %s = map[string]func(%s){\n", *pkg, *name, includeInDispatchMap)
	for _, k := range dispatchMapKeys {
		fmt.Fprintf(
			b,
			"\t%q: %s,\n",
			re.ReplaceAllStringFunc(k, func(s string) string {
				return fmt.Sprintf("-%s", strings.ToLower(s))
			}),
			k,
		)
	}
	fmt.Fprint(b, "}\n")
	p, err := format.Source(b.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(*out, p, 0666); err != nil {
		log.Fatal(err)
	}

}
