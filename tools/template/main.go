package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

func init() {
	log.SetFlags(0)
}

func main() {
	name := flag.String("name", "Template", "function (or, with -receiver-type, method) name which will return this template")
	out := flag.String("o", "", "filename where generated Go code will be written (defaults to appending \".go\" to the input filename)")
	pkg := flag.String("package", "", "package name for the generated Go code (defaults to the name of the directory in which the file is written)")
	receiverType := flag.String("receiver-type", "", "type to which this template should be attached as a method")
	flag.Parse()
	if flag.NArg() > 1 {
		log.Fatal("too many arguments")
	}
	if *out == "" {
		*out = filepath.Base(flag.Arg(0)) + ".go"
	}
	if *pkg == "" {
		abs, err := filepath.Abs(*out)
		if err != nil {
			log.Fatal(err)
		}
		*pkg = filepath.Base(filepath.Dir(abs))
	}

	fi, err := os.Stat(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	var content, returnType string
	if fi.IsDir() {

		content = "map[string]string{\n"
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		filenames, err := f.Readdirnames(-1)
		if err != nil {
			log.Fatal(err)
		}
		sort.Strings(filenames)
		var max int
		for _, filename := range filenames {
			if i := len(filename); i > max {
				max = i
			}
		}
		for _, filename := range filenames {

			// Act a bit like gitignore(5) by at least omitting frequent
			// offenders.
			if strings.HasSuffix(filename, ".swo") ||
				strings.HasSuffix(filename, ".swp") ||
				strings.HasSuffix(filename, ".swx") ||
				strings.HasSuffix(filename, ".zip") {
				continue
			}

			content += fmt.Sprintf(
				"\t\t%q: %s%s,\n",
				filename,
				strings.Repeat(" ", max-len(filename)),
				readFile(filepath.Join(flag.Arg(0), filename)),
			)
		}
		content += "\t}"
		returnType = "map[string]string"

	} else {

		content = readFile(flag.Arg(0))
		returnType = "string"

	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	tmpl := template.Must(template.New("generator").Parse(`package {{.Package}}

// managed by go generate; do not edit by hand

func {{if .ReceiverType}}({{.ReceiverType}}) {{end}}{{.Name}}() {{.ReturnType}} {
	return {{.Content}}
}
`))
	if err := tmpl.Execute(f, struct {
		Content, Name, Package, ReceiverType, ReturnType string
	}{
		Content:      content,
		Name:         *name,
		Package:      *pkg,
		ReceiverType: *receiverType,
		ReturnType:   returnType,
	}); err != nil {
		log.Fatal(err)
	}

}

func readFile(pathname string) string {
	b, err := ioutil.ReadFile(pathname)
	if err != nil {
		log.Fatal(err)
	}
	if bytes.ContainsRune(b, '`') {
		return fmt.Sprintf("%#v", string(b))
	}
	return "`" + string(b) + "`"
}
