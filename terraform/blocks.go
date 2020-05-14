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

type Block interface {
	Ref() Value
	Template() string
}

type Blocks struct {
	blocks []Block
}

func NewBlocks() *Blocks {
	return &Blocks{make([]Block, 0)}
}

func (b *Blocks) Len() int { return len(b.blocks) }

func (b *Blocks) Less(i, j int) bool {
	return b.blocks[i].Ref().Raw() < b.blocks[j].Ref().Raw()
}

func (b *Blocks) Push(block Block) {
	b.blocks = append(b.blocks, block)
}

func (b *Blocks) PushAll(otherBlocks Blocks) {
	for _, block := range otherBlocks.blocks {
		b.Push(block)
	}
}

func (b *Blocks) Swap(i, j int) {
	tmp := b.blocks[i]
	b.blocks[i] = b.blocks[j]
	b.blocks[j] = tmp
}

func (b *Blocks) Write(pathname string) (err error) {

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

	sort.Sort(b)
	for _, block := range b.blocks {
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
