package cmdutil

import "github.com/src-bin/substrate/veqp"

func QualityForEnvironment(environment string) string {
	doc, err := veqp.ReadDocument()
	if err != nil {
		return ""
	}
	return qualityForEnvironment(environment, doc)
}

func qualityForEnvironment(environment string, doc *veqp.Document) (quality string) {
	for _, eq := range doc.ValidEnvironmentQualityPairs {
		if environment == eq.Environment {
			if quality == "" {
				quality = eq.Quality // a candidate but it has to be the only one
			} else {
				return "" // two candidates so no definitive answer
			}
		}
	}
	return quality
}
