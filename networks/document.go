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
	IPv4SubnetMaskLength = 18 // 16,384 IP addresses per VPC, 1,092 possible VPCs
)

type Document struct {
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

func (d *Document) Len() int { return len(d.Networks) }
func (d *Document) Less(i, j int) bool {
	iArray, err := d.Networks[i].comparable()
	if err != nil {
		panic(err)
	}
	jArray, err := d.Networks[j].comparable()
	if err != nil {
		panic(err)
	}
	for k := 0; k < 5; k++ {
		if iArray[k] < jArray[k] {
			return true
		}
		if iArray[k] > jArray[k] {
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

func (d *Document) Next() (*Network, error) {
	if len(d.Networks) == 0 {
		n := &Network{IPv4: fmt.Sprintf("10.0.0.0/%d", IPv4SubnetMaskLength)}
		d.Networks = append(d.Networks, n)
		return n, nil
	}
	sort.Sort(d)
	c, err := d.Networks[len(d.Networks)-1].comparable()
	if err != nil {
		return nil, err
	}
	c[4] = IPv4SubnetMaskLength
	c[3] = 0
	c[2] = c[2] + 64 // TODO parameterize based on IPv4SubnetMaskLength
	if c[2] == 256 {
		c[2] = 0
		c[1] = c[1] + 1
		if c[1] == 256 {
			return nil, errors.New("ran out of /18 networks in 10.0.0.0/8; add support for 172.16.0.0/12 and 192.168.0.0/16")
		}
	}
	c[0] = 10
	n := newNetworkFromComparable(c)
	d.Networks = append(d.Networks, n)
	return n, nil
}

func (d *Document) Write() error {
	b, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(Filename, b, 0666)
}

type Network struct {
	IPv4, IPv6                    string
	Region                        string
	Environment, Quality, Special string `json:",omitempty"`
}

func newNetworkFromComparable(c [5]int) *Network {
	return &Network{IPv4: fmt.Sprintf("%d.%d.%d.%d/%d", c[0], c[1], c[2], c[3], c[4])}
}

func (n *Network) String() string {
	return fmt.Sprintf("%+v", *n) // without dereferencing here, the program OOMs; bizarre
}

func (n *Network) comparable() (c [5]int, err error) {
	fields := strings.FieldsFunc(n.IPv4, func(r rune) bool { return r == '.' || r == '/' })
	if len(fields) != len(c) {
		err = fmt.Errorf("malformed IPv4 %s", n.IPv4)
		return
	}
	for i := 0; i < len(c); i++ {
		c[i], err = strconv.Atoi(fields[i])
		if err != nil {
			return
		}
	}
	return
}

type substrateVersion string

func (v substrateVersion) MarshalJSON() ([]byte, error) {
	if v == "" {
		v = substrateVersion(version.Version)
	}
	return []byte(fmt.Sprintf("%#v", v)), nil
}
