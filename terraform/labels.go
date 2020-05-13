package terraform

import (
	"fmt"
	"strings"
)

// Label performs a domain-specific dimensionality reduction that "feels right"
// enough to represent a resource with a unique label to satisfy Terraform.
func Label(tags Tags, region string, suffixes ...string) Value {
	var s string

	// TODO lots more permutations of tags to support
	if tags.Name != "" {
		s = tags.Name
	} else if tags.Environment != "" && tags.Quality != "" {
		s = fmt.Sprintf("%s-%s", tags.Environment, tags.Quality)
	} else if tags.Special != "" {
		s = tags.Special // TODO possibly deprecate this psuedo-tag in favor of Name
	} else {
		s = "unnamed"
	}

	if tags.AvailabilityZone != "" {
		s = fmt.Sprintf("%s-%s", s, tags.AvailabilityZone)
	} else if region != "" {
		s = fmt.Sprintf("%s-%s", s, region)
	}

	if len(suffixes) > 0 {
		s = fmt.Sprintf("%s-%s", s, strings.Join(suffixes, "-"))
	}

	return Q(s)
}
