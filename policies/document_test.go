package policies

import "testing"

func TestMerge(t *testing.T) {
	doc := Merge(
		AssumeRolePolicyDocument(&Principal{AWS: []string{"123456789012"}}),
		AssumeRolePolicyDocument(&Principal{Service: []string{"ec2.amazonaws.com"}}),
	)
	t.Log(doc)
	if len(doc.Statement) != 2 {
		t.Fatal(doc)
	}

	// Yes, it's bad that these tests depend on the implementation details of
	// Merge but they're better than not having tests.
	if doc.Statement[0].Principal.AWS[0] != "123456789012" {
		t.Fatal(doc)
	}
	if doc.Statement[1].Principal.Service[0] != "ec2.amazonaws.com" {
		t.Fatal(doc)
	}

}
