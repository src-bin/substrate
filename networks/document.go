package networks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/version"
)

const (
	Filename             = "substrate.networks.json"
	IPv4SubnetMaskLength = 18 // 16,384 IP addresses per VPC, 1,092 possible VPCs; value must be in range [16,24]
)

type Document struct {
	Admonition       admonition `json:"#"`
	Networks         []*Network
	SubstrateVersion substrateVersion
}

func ReadDocument() (*Document, error) {
	b, err := fileutil.ReadFile(Filename)
	if errors.Is(err, os.ErrNotExist) {
		b = []byte("{}")
		err = nil
	}
	if err != nil {
		return nil, err
	}
	d := &Document{}

	// If d.SubstrateVersion != version.Version, migrate here.

	d.SubstrateVersion = substrateVersion(version.Version)
	if err := json.Unmarshal(b, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Document) Allocate(n0 *Network) (*Network, error) {
	if n := d.Find(n0); n != nil {
		return n, nil
	}
	return d.next(n0)
}

func (d *Document) Find(n0 *Network) *Network {
	for _, n := range d.Networks {
		if n0.Environment == n.Environment &&
			n0.Quality == n.Quality &&
			n0.Region == n.Region &&
			n0.Special == n.Special {
			return n
		}
	}
	return nil
}

func (d *Document) Len() int { return len(d.Networks) }

func (d *Document) Less(i, j int) bool {
	for k := 0; k < 5; k++ {
		if d.Networks[i].IPv4[k] < d.Networks[j].IPv4[k] {
			return true
		}
		if d.Networks[i].IPv4[k] > d.Networks[j].IPv4[k] {
			return false
		}
	}
	return false
}

func (d *Document) Swap(i, j int) {
	tmp := d.Networks[i]
	d.Networks[i] = d.Networks[j]
	d.Networks[j] = tmp
}

func (d *Document) Write() error {
	b, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(Filename, b, 0666)
}

func (d *Document) next(n *Network) (*Network, error) {
	if len(d.Networks) == 0 {
		n.IPv4 = FirstIPv4()
		d.Networks = append(d.Networks, n)
		return n, nil
	}
	sort.Sort(d)
	var err error
	n.IPv4, err = NextIPv4(d.Networks[len(d.Networks)-1].IPv4)
	if err != nil {
		return nil, err
	}
	d.Networks = append(d.Networks, n)
	return n, d.Write()
}

type IPv4 [5]int

func FirstIPv4() IPv4 {
	return IPv4{10, 0, 0, 0, IPv4SubnetMaskLength}
}

func NextIPv4(ipv4 IPv4) (IPv4, error) {
	if ipv4[4] != IPv4SubnetMaskLength {
		return ipv4, fmt.Errorf("subnet mask %d != IPv4SubnetMaskLength (%d)", ipv4[4], IPv4SubnetMaskLength)
	}
	ipv4[3] = 0
	ipv4[2] = ipv4[2] + (1 << (24 - IPv4SubnetMaskLength)) // this is why IPv4SubnetMaskLength must be in [16, 24]
	if ipv4[2] == 256 {
		ipv4[2] = 0
		ipv4[1] = ipv4[1] + 1
	}
	if ipv4[1] == 256 {
		return ipv4, errors.New("ran out of /18 networks in 10.0.0.0/8; add support for 172.16.0.0/12 and 192.168.0.0/16")
	}
	ipv4[0] = 10
	return ipv4, nil
}

func (ipv4 IPv4) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%#v", ipv4.String())), nil
}

func (ipv4 IPv4) String() string {
	return fmt.Sprintf("%d.%d.%d.%d/%d", ipv4[0], ipv4[1], ipv4[2], ipv4[3], ipv4[4])
}

func (ipv4 *IPv4) UnmarshalJSON(b []byte) (err error) {
	fields := strings.FieldsFunc(
		strings.Trim(string(b), `"`),
		func(r rune) bool { return r == '.' || r == '/' },
	)
	if len(fields) != len(ipv4) {
		return fmt.Errorf("malformed IPv4 %s", string(b))
	}
	for i := 0; i < len(ipv4); i++ {
		ipv4[i], err = strconv.Atoi(fields[i])
		if err != nil {
			return err
		}
	}
	return nil
}

type Network struct {
	Region                        string
	Environment, Quality, Special string `json:",omitempty"`
	IPv4                          IPv4
	IPv6                          string `json:",omitempty"`
	VPC                           string `json:",omitempty"`
}

func (n *Network) String() string {
	return fmt.Sprintf("%+v", *n) // without dereferencing here, the program OOMs; bizarre
}

type admonition struct{}

func (admonition) MarshalJSON() ([]byte, error) {
	return []byte(`"managed by Substrate and synchronized with AWS via Terraform; do not edit by hand"`), nil
}

func (admonition) UnmarshalJSON([]byte) error { return nil }

type substrateVersion string

func (v substrateVersion) MarshalJSON() ([]byte, error) {
	if v == "" {
		v = substrateVersion(version.Version)
	}
	return []byte(fmt.Sprintf("%#v", v)), nil
}

func (substrateVersion) UnmarshalJSON([]byte) error { return nil }
