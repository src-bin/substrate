package naming

import (
	"os"

	"github.com/src-bin/substrate/fileutil"
)

const (
	EnvironmentsFilename = "substrate.environments"
	QualitiesFilename    = "substrate.qualities"

	Default = "default"
)

func Environments() ([]string, error) {
	pathname, err := fileutil.PathnameInParents(EnvironmentsFilename)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(pathname)
	if err != nil {
		return nil, err
	}
	environments := fileutil.ToLines(b)

	// If the file doesn't contain "admin" as one of the environments, add it
	// to the list anyway because it really is still required.
	for _, environment := range environments {
		if environment == Admin {
			return environments, nil
		}
	}
	//log.Printf(`adding "admin" to the list of environments %+v to return %+v`, environments, append([]string{Admin}, environments...))
	return append([]string{Admin}, environments...), nil

}

func Qualities() ([]string, error) {
	pathname, err := fileutil.PathnameInParents(QualitiesFilename)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(pathname)
	if err != nil {
		return nil, err
	}
	return fileutil.ToLines(b), nil
}
