package terraform

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/src-bin/substrate/ui"
)

type Directory struct {
	files map[string]string
}

func NewDirectory() *Directory {
	return &Directory{make(map[string]string)}
}

// TODO func (d *Directory) Add(filename, f *File)

func (d *Directory) AddStatic(filename, content string) {
	d.files[filename] = content
}

func (d *Directory) Write(dirname string) error {

	if err := os.MkdirAll(dirname, 0777); err != nil {
		return err
	}

	for filename, content := range d.files {
		if err := writeFile(dirname, filename, content); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(dirname, filename, content string) (err error) {
	var fp *os.File
	fp, err = ioutil.TempFile(dirname, filename)
	if err != nil {
		return
	}
	if _, err = fp.Write([]byte("# managed by Substrate; do not edit by hand\n\n")); err != nil {
		goto Error
	}
	if _, err = fp.Write([]byte(content)); err != nil {
		goto Error
	}

Error:
	if err := fp.Close(); err != nil {
		log.Print(err)
	}
	pathname := filepath.Join(dirname, filename)
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
