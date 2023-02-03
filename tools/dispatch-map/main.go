package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/src-bin/substrate/fileutil"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/imports"
)

const Main = "Main"

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	out := flag.String("o", "dispatch-map.go", "filename where generated Go code will be written (defaults to \"dispatch-map.go\")")
	pkg := flag.String("package", "main", "package name for the generated Go code (defaults to \"main\")")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("too few arguments")
	}
	if flag.NArg() > 1 {
		log.Fatal("too many arguments")
	}

	// Extract the current package path from go.mod and the current working
	// directory so we can build import statements.
	dirname, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	pathname, err := fileutil.PathnameInParents("go.mod")
	if err != nil {
		log.Fatal(err)
	}
	var basename, pkgPath string
	for i := 0; i < strings.Count(pathname, "../"); i++ {
		dirname, basename = filepath.Split(filepath.Clean(dirname))
		pkgPath = filepath.Join(basename, pkgPath)
	}
	data, err := fileutil.ReadFile(pathname)
	if err != nil {
		log.Fatal(err)
	}
	mod, err := modfile.Parse(pathname, data, nil)
	if err != nil {
		log.Fatal(err)
	}
	pkgPath = filepath.Clean(filepath.Join(filepath.Join(mod.Module.Mod.Path, pkgPath), flag.Arg(0)))
	//log.Print(pkgPath)

	// Look for packages that export a Main function. Make a note of all its
	// parameter types. It's presumed they're all the same; the compiler will
	// catch it if this isn't actually true.
	entries, err := os.ReadDir(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	var dirnames, params, results []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		//log.Printf("%+v", entry)
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, filepath.Join(flag.Arg(0), entry.Name()), nil, parser.ParseComments)
		if err != nil {
			log.Fatal(err)
		}
		if len(pkgs) > 1 {
			log.Fatalf("unexpectedly found more than one package in a single directory (%s)", entry.Name())
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for name, object := range file.Scope.Objects {
					if name == Main && object.Kind == ast.Fun {
						dirnames = append(dirnames, entry.Name())
						if object.Decl.(*ast.FuncDecl).Type.Params != nil {
							params = typeListString(object.Decl.(*ast.FuncDecl).Type.Params.List)
						}
						if object.Decl.(*ast.FuncDecl).Type.Results != nil {
							results = typeListString(object.Decl.(*ast.FuncDecl).Type.Results.List)
						}
					}
				}
			}
		}
	}
	//log.Printf("%+v", params)
	//log.Printf("%+v", results)

	// Generate and format Go code that declares the dispatch map. Rewrite
	// camelCase function names in dash-case to match command-line arguments.
	b := &bytes.Buffer{}
	re := regexp.MustCompile(`[\p{Lu}]`)
	fmt.Fprintf(b, "package %s\n\nimport (\n", *pkg)
	for _, dirname := range dirnames {
		if strings.Contains(dirname, "-") {

			// Remove dashes from directory names as is convention for package names.
			if dirnameNoDashes := strings.ReplaceAll(dirname, "-", ""); dirnameNoDashes != *pkg {
				fmt.Fprintf(b, "\t%s \"%s/%s\"\n", dirnameNoDashes, pkgPath, dirname)
			}

		} else if dirname != *pkg {
			fmt.Fprintf(b, "\t\"%s/%s\"\n", pkgPath, dirname)
		}
	}
	joinedParams := strings.Join(params, ", ")
	joinedResults := strings.Join(results, ", ")
	switch len(results) {
	case 0:
		fmt.Fprintf(b, ")\n\nvar dispatchMap = map[string]func(%s){\n", joinedParams)
	case 1:
		fmt.Fprintf(b, ")\n\nvar dispatchMap = map[string]func(%s) %s {\n", joinedParams, joinedResults)
	default:
		fmt.Fprintf(b, ")\n\nvar dispatchMap = map[string]func(%s) (%s) {\n", joinedParams, joinedResults)
	}
	for _, dirname := range dirnames {

		// Turn camelCase and snake_case into dash-case for the command-line argument.
		// (This is probably superfluous.)
		subcommand := re.ReplaceAllStringFunc(dirname, func(s string) string {
			return fmt.Sprintf("-%s", strings.ReplaceAll(strings.ToLower(s), "_", "-"))
		})

		dirnameNoDashes := strings.ReplaceAll(dirname, "-", "")
		if dirnameNoDashes == *pkg {
			fmt.Fprintf(
				b,
				"\t%q: func(context.Context, *awscfg.Config) {},\n",
				subcommand,
			)
		} else {
			fmt.Fprintf(
				b,
				"\t%q: %s.%s,\n",
				subcommand,
				strings.ReplaceAll(dirname, "-", ""),
				Main,
			)
		}
	}
	fmt.Fprint(b, "}\n")
	p, err := imports.Process(*out, b.Bytes(), nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(*out, p, 0666); err != nil {
		log.Fatal(err)
	}

}

func typeListString(list []*ast.Field) []string {
	strings := []string{}
	for _, item := range list {
		s := ""
		t := item.Type
		//log.Printf("%T %+v", t, t)
		if i, ok := t.(*ast.Ident); ok {
			s = i.Name
		}
		if se, ok := t.(*ast.StarExpr); ok {
			s = "*"
			t = se.X
		}
		if se, ok := t.(*ast.SelectorExpr); ok {
			s = fmt.Sprintf("%s%s.%s", s, se.X, se.Sel)
		}
		strings = append(strings, s)
	}
	return strings
}
