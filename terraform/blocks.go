package terraform

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"text/template"

	"github.com/src-bin/substrate/ui"
)

type Block interface {
	Template() string
}

type Blocks struct {
	blocks []Block
}

func NewBlocks() *Blocks {
	return &Blocks{make([]Block, 0)}
}

func (blocks *Blocks) Push(block Block) {
	blocks.blocks = append(blocks.blocks, block)
}

func (blocks *Blocks) Write(pathname string) (err error) {
	dirname := path.Dir(pathname)
	if err = os.MkdirAll(dirname, 0777); err != nil {
		return
	}
	var f *os.File
	f, err = ioutil.TempFile(dirname, path.Base(pathname))
	if err != nil {
		return
	}
	fmt.Fprintln(f, "# managed by Substrate; do not edit by hand")
	for _, block := range blocks.blocks {
		fmt.Fprintln(f, "")
		var tmpl *template.Template
		tmpl, err = template.New(fmt.Sprintf("%T", block)).Parse(block.Template())
		if err != nil {
			goto Error
		}
		if err = tmpl.Execute(f, block); err != nil {
			goto Error
		}
		fmt.Fprintln(f, "")
	}
Error:
	if err := f.Close(); err != nil {
		log.Print(err)
	}
	if err == nil {
		err = os.Rename(f.Name(), pathname)
	} else {
		if err := os.Remove(f.Name()); err != nil {
			log.Print(err)
		}
	}
	ui.Printf("wrote %s", pathname)
	return
}
