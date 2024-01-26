package networks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/src-bin/substrate/cidr"
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
	RFC1918              cidr.IPv4
	SubstrateVersion     jsonutil.SubstrateVersion
	filename             string
}

func ReadDocument(filename string, rfc1918 cidr.IPv4, subnetMaskLength int) (*Document, error) {
	b, err := os.ReadFile(filename)
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

	zero := cidr.IPv4{0, 0, 0, 0, 0}
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
		n.IPv4 = cidr.FirstIPv4(d.RFC1918, d.IPv4SubnetMaskLength)
		d.Networks = append(d.Networks, n)
		return n, nil
	}
	sort.Sort(d)
	var err error

	// TODO Tolerate varied CIDR prefix lengths. If there's a long prefix that
	// sorts into the middle of the list then we might produce overlapping
	// entries. If there's a long prefix that's not aligned with its length
	// (unlikely thought it would be for someone to manually add that), we'll
	// happily produce nonsensical entries with octets greater than 255.
	n.IPv4, err = cidr.NextIPv4(d.Networks[len(d.Networks)-1].IPv4)

	if err != nil {
		return nil, err
	}
	d.Networks = append(d.Networks, n)
	return n, d.Write()
}

type Network struct {
	Region                        string
	Environment, Quality, Special string `json:",omitempty"`
	IPv4                          cidr.IPv4
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
