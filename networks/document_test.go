package networks

import (
	"testing"

	"github.com/src-bin/substrate/cidr"
)

func TestFirstAndNextIPv4(t *testing.T) {
	ipv4 := cidr.FirstIPv4(cidr.RFC1918_10_0_0_0_8, 18)
	if ipv4.String() != "10.0.0.0/18" {
		t.Fatal(ipv4)
	}

	ipv4, err := cidr.NextIPv4(ipv4)
	if err != nil {
		t.Fatal(err)
	}
	if ipv4.String() != "10.0.64.0/18" {
		t.Fatal(ipv4)
	}

	ipv4, err = cidr.NextIPv4(ipv4)
	if err != nil {
		t.Fatal(err)
	}
	if ipv4.String() != "10.0.128.0/18" {
		t.Fatal(ipv4)
	}
}

func TestMixedSubnetLength(t *testing.T) {
	d := &Document{
		IPv4SubnetMaskLength: 18,
		Networks: []*Network{
			&Network{IPv4: cidr.IPv4{10, 0, 0, 0, 18}},
		},
		RFC1918: cidr.IPv4{10, 0, 0, 0, 8},
	}

	d.Ensure(&Network{Environment: "test1"})
	if d.Networks[1].IPv4 != (cidr.IPv4{10, 0, 64, 0, 18}) {
		t.Fatal(d)
	}

	// Simulate manually adding an entry.
	d.Networks = append(d.Networks, &Network{IPv4: cidr.IPv4{10, 1, 0, 0, 16}})

	d.Ensure(&Network{Environment: "test2"})
	if d.Networks[3].IPv4 != (cidr.IPv4{10, 2, 0, 0, 16}) {
		t.Fatal(d)
	}
}
