package fileutil

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// Download requests the given URL with an HTTP GET request and stores it
// in a temporary file named per the given pattern. (See os.CreateTemp for
// the rules.)
func Download(u *url.URL, pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if err := download(u, f); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func download(u *url.URL, f *os.File) error {
	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("<%s> responded %s", u.String(), resp.Status)
	}
	if _, err = io.Copy(f, resp.Body); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}
