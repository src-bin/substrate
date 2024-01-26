package cidr

import (
	"encoding/json"
	"testing"
)

func TestParseIPv6(t *testing.T) {
	if _, err := ParseIPv6("malformed"); err == nil {
		t.Fatal(`expected error from ParseIPv6("malformed")`)
	}
	if ipv6, err := ParseIPv6("::/0"); ipv6 != (IPv6{}) || err != nil {
		t.Fatal(ipv6, err)
	}
	if ipv6, err := ParseIPv6("2600:1f13:31a:3600::/56"); ipv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 0, 0, 0, 0, 0, 0, 0, 0, 0, 56}) || err != nil {
		t.Fatal(ipv6, err)
	}
}

func TestSubnetIPv6(t *testing.T) {
	ipv6 := MustIPv6(ParseIPv6("2600:1f13:31a:3600::/56"))

	// In a network with only public subnets, this one's wasted. In a network
	// with public and private subnets, this one's further divided into the
	// public subnets.
	if subnetIPv6, err := ipv6.SubnetIPv6(2, 0); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 0, 0, 0, 0, 0, 0, 0, 0, 0, 58}) {
		t.Fatal(subnetIPv6, err)
	}

	// The first subnet is wasted.
	if subnetIPv6, err := ipv6.SubnetIPv6(4, 0); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 0, 0, 0, 0, 0, 0, 0, 0, 0, 60}) {
		t.Fatal(subnetIPv6, err)
	}

	// Public subnets.
	if subnetIPv6, err := ipv6.SubnetIPv6(4, 1); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 16, 0, 0, 0, 0, 0, 0, 0, 0, 60}) {
		t.Fatal(subnetIPv6, err)
	}
	if subnetIPv6, err := ipv6.SubnetIPv6(4, 2); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 32, 0, 0, 0, 0, 0, 0, 0, 0, 60}) {
		t.Fatal(subnetIPv6, err)
	}
	if subnetIPv6, err := ipv6.SubnetIPv6(4, 3); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 48, 0, 0, 0, 0, 0, 0, 0, 0, 60}) {
		t.Fatal(subnetIPv6, err)
	}

	// Mathematically legal but unused by Substrate. This space is used by a
	// larger private subnet.
	if subnetIPv6, err := ipv6.SubnetIPv6(4, 4); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 64, 0, 0, 0, 0, 0, 0, 0, 0, 60}) {
		t.Fatal(subnetIPv6, err)
	}

	// Private subnets.
	if subnetIPv6, err := ipv6.SubnetIPv6(2, 1); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 64, 0, 0, 0, 0, 0, 0, 0, 0, 58}) {
		t.Fatal(subnetIPv6, err)
	}
	if subnetIPv6, err := ipv6.SubnetIPv6(2, 2); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 128, 0, 0, 0, 0, 0, 0, 0, 0, 58}) {
		t.Fatal(subnetIPv6, err)
	}
	if subnetIPv6, err := ipv6.SubnetIPv6(2, 3); err != nil || subnetIPv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 192, 0, 0, 0, 0, 0, 0, 0, 0, 58}) {
		t.Fatal(subnetIPv6, err)
	}

	// Mathematically invalid subnet given the number of bits we have to identify them.
	if subnetIPv6, err := ipv6.SubnetIPv6(2, 4); err == nil || err.Error() != "subnet 4 outside 2-bit range [0, 4)" {
		t.Fatal(subnetIPv6, err)
	}

}

func TestUnmarshalIPv6(t *testing.T) {
	var ipv6 IPv6
	if err := json.Unmarshal([]byte(`"malformed"`), &ipv6); err == nil {
		t.Fatal(`expected error from unmarshaling "malformed" into IPv6`)
	}
	if err := json.Unmarshal([]byte(`"::/0"`), &ipv6); ipv6 != (IPv6{}) || err != nil {
		t.Fatal(ipv6, err)
	}
	if err := json.Unmarshal([]byte(`"2600:1f13:31a:3600::/56"`), &ipv6); ipv6 != (IPv6{38, 0, 31, 19, 3, 26, 54, 0, 0, 0, 0, 0, 0, 0, 0, 0, 56}) || err != nil {
		t.Fatal(ipv6, err)
	}
}
