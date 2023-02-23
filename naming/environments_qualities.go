package naming

import "github.com/src-bin/substrate/fileutil"

const (
	EnvironmentsFilename = "substrate.environments"
	QualitiesFilename    = "substrate.qualities"
)

func Environments() ([]string, error) {
	pathname, err := fileutil.PathnameInParents(EnvironmentsFilename)
	if err != nil {
		return nil, err
	}
	b, err := fileutil.ReadFile(pathname)
	if err != nil {
		return nil, err
	}
	return fileutil.ToLines(b), nil
}

func Qualities() ([]string, error) {
	pathname, err := fileutil.PathnameInParents(QualitiesFilename)
	if err != nil {
		return nil, err
	}
	b, err := fileutil.ReadFile(pathname)
	if err != nil {
		return nil, err
	}
	return fileutil.ToLines(b), nil
}
