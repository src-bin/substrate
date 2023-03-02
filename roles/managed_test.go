package roles

import "testing"

func TestManagedAssumeRolePolicyZero(t *testing.T) {
	p := &ManagedAssumeRolePolicy{}
	if len(p.Arguments()) != 0 {
		t.Errorf("len(p.Arguments()): %d != 0; p.Arguments(): %+v", len(p.Arguments()), p.Arguments())
	}
	if p.String() != "" {
		t.Errorf(`p.String(): %q != ""`, p.String())
	}
}

func TestManagedPolicyAttachmentsZero(t *testing.T) {
	a := &ManagedPolicyAttachments{}
	if len(a.Arguments()) != 0 {
		t.Errorf("len(a.Arguments()): %d != 0; a.Arguments(): %+v", len(a.Arguments()), a.Arguments())
	}
	if a.String() != "" {
		t.Errorf(`a.String(): %q != ""`, a.String())
	}
}
