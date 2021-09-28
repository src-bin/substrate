package awssessions

import (
	"errors"
	"testing"
)

func TestOrganizationReaderError(t *testing.T) {
	if err := NewOrganizationReaderError(errors.New("original"), ""); err.Error() != "could not assume the OrganizationReader role in your organization's management account, which is a prerequisite for finding and assuming other roles (actual error: original)" {
		t.Error(err)
	}
	if err := NewOrganizationReaderError(errors.New("original"), "test"); err.Error() != "could not assume the OrganizationReader role in your organization's management account, which is a prerequisite for finding and assuming the test role (actual error: original)" {
		t.Error(err)
	}
}
