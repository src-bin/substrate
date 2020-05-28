package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
		*out = flag.Arg(0) + ".go"
	}
	if *pkg == "" {
		abs, err := filepath.Abs(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		*pkg = filepath.Base(filepath.Dir(abs))
	}

	b, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	var content string
	if bytes.ContainsRune(b, '`') {
		content = fmt.Sprintf("%#v", string(b))
	} else {
		content = "`" + string(b) + "`"
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	tmpl := template.Must(template.New("generator").Parse(`package {{.Package}}

// managed by go generate; do not edit by hand

func {{if .ReceiverType}}({{.ReceiverType}}) {{end}}{{.Name}}() string {
	return {{.Content}}
}
`))
	if err := tmpl.Execute(f, struct {
		Content, Name, Package, ReceiverType string
	}{
		Content:      content,
		Name:         *name,
		Package:      *pkg,
		ReceiverType: *receiverType,
	}); err != nil {
		log.Fatal(err)
	}

}
