package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/src-bin/substrate/fileutil"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/imports"
)

type Map struct {
	Func bool // true to include a Func pointer in the dispatch map
	Map  map[string]*Map
}

func (m *Map) Add(ss []string) {
	if len(ss) == 0 {
		m.Func = true
		return
	}
	if m.Map == nil {
		m.Map = make(map[string]*Map)
	}
	if m.Map[ss[0]] == nil {
		m.Map[ss[0]] = &Map{}
	}
	m.Map[ss[0]].Add(ss[1:])
}

func (m *Map) Write(funcName, funcType string, w io.Writer) {
	m.write("", "", funcName, funcType, w)
}

func (m *Map) write(indent, dirname, funcName, funcType string, w io.Writer) {
	fmt.Fprintf(w, "&dispatchMap%s{\n", funcName) // no indent because this is an expression
	if dirname != "" && m.Func {
		fmt.Fprintf(w, "%s\tFunc: %s.%s,\n", indent, pkgName(dirname), funcName)
	}
	if m.Map != nil {
		fmt.Fprintf(w, "%s\tMap: map[string]*dispatchMap%s{\n", indent, funcName)
		re := regexp.MustCompile(`[\p{Lu}]`)
		for s, m2 := range m.Map {

			// Turn camelCase and snake_case directory names into dash-case
			// subcommand names. (This is probably superfluous because I don't
			// actually name directories in camelCase or snake_case.)
			fmt.Fprintf(w, "%s\t\t%q: ", indent, re.ReplaceAllStringFunc(s, func(s string) string {
				return fmt.Sprintf("-%s", strings.ReplaceAll(strings.ToLower(s), "_", "-"))
			}))

			m2.write(indent+"\t\t", filepath.Join(dirname, s), funcName, funcType, w)
			fmt.Fprintf(w, ",\n")
		}
		fmt.Fprintf(w, "%s\t},\n", indent)
	}
	fmt.Fprintf(w, "%s}", indent)
}

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	out := flag.String("o", "dispatch-map.go", "filename where generated Go code will be written (defaults to \"dispatch-map.go\")")
	pkg := flag.String("package", "main", "package name for the generated Go code (defaults to \"main\")")
	function := flag.String("function", "Main", "function name to look for in each package (defaults to \"Main\")")
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
	data, err := os.ReadFile(pathname)
	if err != nil {
		log.Fatal(err)
	}
	mod, err := modfile.Parse(pathname, data, nil)
	if err != nil {
		log.Fatal(err)
	}
	pkgPath = filepath.Clean(filepath.Join(filepath.Join(mod.Module.Mod.Path, pkgPath), flag.Arg(0)))
	//log.Print(pkgPath)

	// Look for packages that export the function named by the -function
	// argument. Make a note of all its parameter types. It's presumed they're
	// all the same; the compiler will catch it if this isn't actually true.
	var dirnames, params, results []string
	m := &Map{}
	fs.WalkDir(os.DirFS(flag.Arg(0)), ".", func(pathname string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, filepath.Join(flag.Arg(0), pathname), nil, parser.ParseComments)
		if err != nil {
			log.Fatal(err)
		}
		if len(pkgs) > 1 {
			log.Fatalf("unexpectedly found more than one package in a single directory (%s)", entry.Name())
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for name, object := range file.Scope.Objects {
					if name == *function && object.Kind == ast.Fun {
						dirnames = append(dirnames, pathname)
						m.Add(strings.Split(pathname, "/"))
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
		return nil
	})
	//log.Printf("%+v", dirnames)
	//log.Printf("%+v", params)
	//log.Printf("%+v", results)
	//log.Print(jsonutil.MustString(m))

	// Generate and format Go code that declares the dispatch map.
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "package %s\n\nimport (\n", *pkg)
	for _, dirname := range dirnames {
		fmt.Fprintf(b, "\t%s \"%s/%s\"\n", pkgName(dirname), pkgPath, dirname)
	}
	joinedParams := strings.Join(params, ", ")
	joinedResults := strings.Join(results, ", ")
	var funcType string
	switch len(results) {
	case 0:
		funcType = fmt.Sprintf("func(%s)", joinedParams)
	case 1:
		funcType = fmt.Sprintf("func(%s) %s", joinedParams, joinedResults)
	default:
		funcType = fmt.Sprintf("func(%s) (%s)", joinedParams, joinedResults)
	}
	fmt.Fprintf(b, ")\n\n")
	fmt.Fprintf(b, "var DispatchMap%s = ", *function)
	m.Write(*function, funcType, b)
	fmt.Fprintf(b, "\ntype dispatchMap%s struct {\n\tFunc %s\n\tMap map[string]*dispatchMap%s\n}\n\n", *function, funcType, *function)

	//log.Print(string(b.Bytes()))
	p, err := imports.Process(*out, b.Bytes(), nil)
	if err != nil {
		log.Fatal(err)
	}
	//log.Print(string(p))

	if err := os.WriteFile(*out, p, 0666); err != nil {
		log.Fatal(err)
	}
}

// pkgName sanitizes a possibly nested directory path into an identifier that,
// while maybe not idiomatic Go, will definitely work in the generated map.
func pkgName(dirname string) (pkg string) {
	pkg = strings.ToLower(dirname)
	pkg = strings.ReplaceAll(pkg, "-", "")
	pkg = strings.ReplaceAll(pkg, "_", "")
	pkg = strings.ReplaceAll(pkg, "/", "_") // last, after "_" is replaced
	return
}

// typeListString returns a slice of syntactically valid Go type declarations
// as strings to be used in the generated map.
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
