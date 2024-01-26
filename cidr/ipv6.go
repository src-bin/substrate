package cidr

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/src-bin/substrate/ui"
)

type IPv6 [17]uint8

func MustIPv6(ipv6 IPv6, err error) IPv6 {
	if err != nil {
		ui.Fatal(err)
	}
	return ipv6
}

func ParseIPv6(s string) (ipv6 IPv6, err error) {
	err = parseIPv6(s, &ipv6)
	return
}

func (ipv6 IPv6) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%#v", ipv6.String())), nil
}

func (ipv6 IPv6) String() string {
	return fmt.Sprintf("%s/%d", net.IP(ipv6[:16]).String(), ipv6[16])
}

func (ipv6 IPv6) SubnetIPv6(bits, subnet int) (IPv6, error) {
	ipv6[16] += uint8(bits)

	if ipv6[16] < 56 || ipv6[16] > 64 {
		return ipv6, fmt.Errorf("prefix length %d outside range [56, 64]", ipv6[16])
	}

	if subnet < 0 || subnet >= 1<<bits {
		return ipv6, fmt.Errorf("subnet %d outside %d-bit range [0, %d)", subnet, bits, 1<<bits)
	}

	ipv6[15] = 0
	ipv6[14] = 0
	ipv6[13] = 0
	ipv6[12] = 0
	ipv6[11] = 0
	ipv6[10] = 0
	ipv6[9] = 0
	ipv6[8] = 0
	ipv6[7] = ipv6[7] + uint8(subnet)*(1<<(64-ipv6[16])) // relies on prefix length in range [56, 64]
	return ipv6, nil
}

func (ipv6 *IPv6) UnmarshalJSON(b []byte) (err error) {
	return parseIPv6(strings.Trim(string(b), `"`), ipv6)
}

func parseIPv6(s string, ipv6 *IPv6) (err error) {
	fields := strings.Split(s, "/")
	if len(fields) != 2 {
		return fmt.Errorf("malformed IPv6 %s", s)
	}
	copy(ipv6[:16], net.ParseIP(fields[0]).To16())
	var i int
	i, err = strconv.Atoi(fields[1])
	ipv6[16] = uint8(i)
	return
}
