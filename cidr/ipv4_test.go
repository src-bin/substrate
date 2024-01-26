package cidr

import (
	"encoding/json"
	"testing"
)

func TestParseIPv4(t *testing.T) {
	if _, err := ParseIPv4("malformed"); err == nil {
		t.Fatal(`expected error from ParseIPv4("malformed")`)
	}
	if ipv4, err := ParseIPv4("0.0.0.0/0"); ipv4 != (IPv4{}) || err != nil {
		t.Fatal(ipv4, err)
	}
	if ipv4, err := ParseIPv4("1.1.1.1/1"); ipv4 != (IPv4{1, 1, 1, 1, 1}) || err != nil {
		t.Fatal(ipv4, err)
	}
}

func TestSubnetIPv4(t *testing.T) {
	ipv4 := MustIPv4(ParseIPv4("10.0.0.0/18"))

	// In a network with only public subnets, this one's wasted. In a network
	// with public and private subnets, this one's further divided into the
	// public subnets.
	if subnetIPv4, err := ipv4.SubnetIPv4(2, 0); err != nil || subnetIPv4 != (IPv4{10, 0, 0, 0, 20}) {
		t.Fatal(subnetIPv4, err)
	}

	// The first subnet is wasted.
	if subnetIPv4, err := ipv4.SubnetIPv4(4, 0); err != nil || subnetIPv4 != (IPv4{10, 0, 0, 0, 22}) {
		t.Fatal(subnetIPv4, err)
	}

	// Public subnets.
	if subnetIPv4, err := ipv4.SubnetIPv4(4, 1); err != nil || subnetIPv4 != (IPv4{10, 0, 4, 0, 22}) {
		t.Fatal(subnetIPv4, err)
	}
	if subnetIPv4, err := ipv4.SubnetIPv4(4, 2); err != nil || subnetIPv4 != (IPv4{10, 0, 8, 0, 22}) {
		t.Fatal(subnetIPv4, err)
	}
	if subnetIPv4, err := ipv4.SubnetIPv4(4, 3); err != nil || subnetIPv4 != (IPv4{10, 0, 12, 0, 22}) {
		t.Fatal(subnetIPv4, err)
	}

	// Mathematically legal but unused by Substrate. This space is used by a
	// larger private subnet.
	if subnetIPv4, err := ipv4.SubnetIPv4(4, 4); err != nil || subnetIPv4 != (IPv4{10, 0, 16, 0, 22}) {
		t.Fatal(subnetIPv4, err)
	}

	// Private subnets.
	if subnetIPv4, err := ipv4.SubnetIPv4(2, 1); err != nil || subnetIPv4 != (IPv4{10, 0, 16, 0, 20}) {
		t.Fatal(subnetIPv4, err)
	}
	if subnetIPv4, err := ipv4.SubnetIPv4(2, 2); err != nil || subnetIPv4 != (IPv4{10, 0, 32, 0, 20}) {
		t.Fatal(subnetIPv4, err)
	}
	if subnetIPv4, err := ipv4.SubnetIPv4(2, 3); err != nil || subnetIPv4 != (IPv4{10, 0, 48, 0, 20}) {
		t.Fatal(subnetIPv4, err)
	}

	// Mathematically invalid subnet given the number of bits we have to identify them.
	if subnetIPv4, err := ipv4.SubnetIPv4(2, 4); err == nil || err.Error() != "subnet 4 outside 2-bit range [0, 4)" {
		t.Fatal(subnetIPv4, err)
	}

}

func TestUnmarshalIPv4(t *testing.T) {
	var ipv4 IPv4
	if err := json.Unmarshal([]byte(`"malformed"`), &ipv4); err == nil {
		t.Fatal(`expected error from unmarshaling "malformed" into IPv4`)
	}
	if err := json.Unmarshal([]byte(`"0.0.0.0/0"`), &ipv4); ipv4 != (IPv4{}) || err != nil {
		t.Fatal(ipv4, err)
	}
	if err := json.Unmarshal([]byte(`"1.1.1.1/1"`), &ipv4); ipv4 != (IPv4{1, 1, 1, 1, 1}) || err != nil {
		t.Fatal(ipv4, err)
	}
}
