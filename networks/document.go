package networks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/version"
)

const (
	AdminFilename = "substrate.admin-networks.json"
	Filename      = "substrate.networks.json"
)

type Document struct {
	Admonition           jsonutil.Admonition `json:"#"`
	IPv4SubnetMaskLength int                 // must be in range [16, 24]
	Networks             []*Network
	RFC1918              IPv4
	SubstrateVersion     jsonutil.SubstrateVersion
	filename             string
}

func ReadDocument(filename string, rfc1918 IPv4, subnetMaskLength int) (*Document, error) {
	b, err := fileutil.ReadFile(filename)
	if errors.Is(err, fs.ErrNotExist) {
		b = []byte("{}")
		err = nil
	}
	if err != nil {
		return nil, err
	}
	d := &Document{filename: filename}
	if err := json.Unmarshal(b, d); err != nil {
		return nil, err
	}

	// If d.SubstrateVersion != version.Version, migrate here.

	zero := IPv4{0, 0, 0, 0, 0}
	if d.RFC1918 == zero {
		d.RFC1918 = rfc1918
	}
	if d.IPv4SubnetMaskLength == 0 {
		d.IPv4SubnetMaskLength = subnetMaskLength
	}

	d.SubstrateVersion = jsonutil.SubstrateVersion(version.Version)
	return d, nil
}

func (d *Document) Ensure(n0 *Network) (*Network, error) {
	if n := d.Find(n0); n != nil {
		return n, nil
	}
	return d.next(n0)
}

func (d *Document) Find(n0 *Network) *Network {
	for _, n := range d.Networks {
		if match(n0, n) {
			return n
		}
	}
	return nil
}

func (d *Document) FindAll(n0 *Network) (nets []*Network) {
	for _, n := range d.Networks {
		if match(n0, n) {
			nets = append(nets, n)
		}
	}
	return nets
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
	return jsonutil.Write(d, d.filename)
}

func (d *Document) next(n *Network) (*Network, error) {
	if len(d.Networks) == 0 {
		n.IPv4 = FirstIPv4(d.RFC1918, d.IPv4SubnetMaskLength)
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

var (
	RFC1918_10_0_0_0_8     = IPv4{10, 0, 0, 0, 8}
	RFC1918_172_16_0_0_12  = IPv4{172, 16, 0, 0, 12}
	RFC1918_192_168_0_0_16 = IPv4{192, 168, 0, 0, 16}
)

func FirstIPv4(rfc1918 IPv4, subnetMaskLength int) IPv4 {
	rfc1918[4] = subnetMaskLength
	return rfc1918
}

func NextIPv4(ipv4 IPv4) (IPv4, error) {
	if ipv4[4] < 16 || ipv4[4] > 24 {
		return ipv4, fmt.Errorf("subnet mask %d outside range [16, 24]", ipv4[4])
	}
	ipv4[3] = 0
	ipv4[2] = ipv4[2] + (1 << (24 - ipv4[4])) // this is why IPv4SubnetMaskLength must be in [16, 24]
	if ipv4[2] == 256 {
		if ipv4[0] == 192 {
			return ipv4, errors.New("ran out of networks in 192.168.0.0/16")
		}
		ipv4[2] = 0
		ipv4[1] = ipv4[1] + 1
	}
	if ipv4[0] == 10 && ipv4[1] == 256 {
		return ipv4, errors.New("ran out of networks in 10.0.0.0/8")
	}
	if ipv4[0] == 172 && ipv4[1] == 32 {
		return ipv4, errors.New("ran out of networks in 172.16.0.0/12")
	}
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

// match returns true iff every field in n0 that's not empty matches the
// corresponding field in n.
func match(n0, n *Network) bool {
	return (n0.Environment == "" || n0.Environment == n.Environment) &&
		(n0.Quality == "" || n0.Quality == n.Quality) &&
		(n0.Region == "" || n0.Region == n.Region) &&
		(n0.Special == "" || n0.Special == n.Special)
}
