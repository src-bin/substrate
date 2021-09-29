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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/src-bin/substrate/fileutil"
	"golang.org/x/mod/modfile"
)

const Main = "Main"

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	name := flag.String("name", "dispatchMap", "function (or, with -receiver-type, method) name")
	out := flag.String("o", "dispatch-map.go", "filename where generated Go code will be written (defaults to \"dispatch-map.go\")")
	pkg := flag.String("package", "main", "package name for the generated Go code (defaults to \"main\")")
	flag.Parse()
	if flag.NArg() > 0 {
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
	pkgPath = filepath.Join(mod.Module.Mod.Path, pkgPath)

	// Look for packages that export a Main function that accepts zero parameters.
	entries, err := os.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}
	var dirnames []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		//log.Printf("%+v", entry)
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, entry.Name(), nil, parser.ParseComments)
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
						if list := object.Decl.(*ast.FuncDecl).Type.Params.List; len(list) == 0 {
							dirnames = append(dirnames, entry.Name())
						}
					}
				}
			}
		}

	}

	// Generate and format Go code that declares the dispatch map. Rewrite
	// camelCase function names in dash-case to match command-line arguments.
	b := &bytes.Buffer{}
	re := regexp.MustCompile(`[\p{Lu}]`)
	fmt.Fprintf(b, "package %s\n\nimport (\n", *pkg)
	for _, dirname := range dirnames {
		if strings.Contains(dirname, "-") {

			// Remove dashes from directory names as is convention for package names.
			fmt.Fprintf(b, "\t%s \"%s/%s\"\n", strings.ReplaceAll(dirname, "-", ""), pkgPath, dirname)

		} else {
			fmt.Fprintf(b, "\t\"%s/%s\"\n", pkgPath, dirname)
		}

	}
	fmt.Fprintf(b, ")\n\nvar %s = map[string]func(){\n", *name)
	for _, dirname := range dirnames {
		fmt.Fprintf(
			b,
			"\t%q: %s.%s,\n",

			// Turn camelCase and snake_case into dash-case for the command-line argument.
			re.ReplaceAllStringFunc(dirname, func(s string) string {
				return fmt.Sprintf("-%s", strings.ReplaceAll(strings.ToLower(s), "_", "-"))
			}),

			strings.ReplaceAll(dirname, "-", ""),
			Main,
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
