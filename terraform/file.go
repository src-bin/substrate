package terraform

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"text/template"

	"github.com/src-bin/substrate/ui"
)

type File struct {
	blocks []Block
}

func NewFile() *File {
	return &File{make([]Block, 0)}
}

func (f *File) Len() int { return len(f.blocks) }

func (f *File) Less(i, j int) bool {
	return f.blocks[i].Ref().Raw() < f.blocks[j].Ref().Raw()
}

func (f *File) Push(b Block) {
	f.blocks = append(f.blocks, b)
}

func (f *File) PushAll(otherFile *File) {
	for _, b := range otherFile.blocks {
		f.Push(b)
	}
}

func (f *File) Swap(i, j int) {
	tmp := f.blocks[i]
	f.blocks[i] = f.blocks[j]
	f.blocks[j] = tmp
}

func (f *File) Write(pathname string) (err error) {

	dirname := path.Dir(pathname)
	if err = os.MkdirAll(dirname, 0777); err != nil {
		return
	}

	var fp *os.File
	fp, err = ioutil.TempFile(dirname, path.Base(pathname))
	if err != nil {
		return
	}
	fmt.Fprintln(fp, "# managed by Substrate; do not edit by hand")

	sort.Sort(f)
	for _, b := range f.blocks {
		if _, err = fmt.Fprintln(fp, ""); err != nil {
			goto Error
		}
		var tmpl *template.Template
		tmpl, err = template.New(fmt.Sprintf("%T", b)).Parse(b.Template())
		if err != nil {
			goto Error
		}
		if err = tmpl.Execute(fp, b); err != nil {
			goto Error
		}
		if _, err = fmt.Fprintln(fp, ""); err != nil {
			goto Error
		}
	}

Error:
	if err := fp.Close(); err != nil {
		log.Print(err)
	}
	if err == nil {
		err = os.Rename(fp.Name(), pathname)
	} else {
		if err := os.Remove(fp.Name()); err != nil {
			log.Print(err)
		}
	}
	ui.Printf("wrote %s", pathname)
	return
}
