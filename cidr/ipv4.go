package cidr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type IPv4 [5]int // not uint8 because we do math that would wrap

var (
	RFC1918_10_0_0_0_8     = IPv4{10, 0, 0, 0, 8}
	RFC1918_172_16_0_0_12  = IPv4{172, 16, 0, 0, 12}
	RFC1918_192_168_0_0_16 = IPv4{192, 168, 0, 0, 16}
)

func FirstIPv4(rfc1918 IPv4, prefixLength int) IPv4 {
	rfc1918[4] = prefixLength
	return rfc1918
}

func MustParseIPv4(s string) IPv4 {
	ipv4, err := ParseIPv4(s)
	if err != nil {
		panic(err)
	}
	return ipv4
}

func NextIPv4(ipv4 IPv4) (IPv4, error) {
	if ipv4[4] < 16 || ipv4[4] > 24 {
		return ipv4, fmt.Errorf("prefix length %d outside range [16, 24]", ipv4[4])
	}
	ipv4[3] = 0
	ipv4[2] = ipv4[2] + (1 << (24 - ipv4[4])) // relies on prefix length in range [16, 24]
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

func ParseIPv4(s string) (ipv4 IPv4, err error) {
	err = parseIPv4(s, &ipv4)
	return
}

func (ipv4 IPv4) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%#v", ipv4.String())), nil
}

func (ipv4 IPv4) String() string {
	return fmt.Sprintf("%d.%d.%d.%d/%d", ipv4[0], ipv4[1], ipv4[2], ipv4[3], ipv4[4])
}

func (ipv4 IPv4) SubnetIPv4(bits, subnet int) (IPv4, error) {
	ipv4[4] += bits
	if ipv4[4] < 16 || ipv4[4] > 24 {
		return ipv4, fmt.Errorf("prefix length %d outside range [16, 24]", ipv4[4])
	}
	if subnet < 0 || subnet >= 1<<bits {
		return ipv4, fmt.Errorf("subnet %d outside %d-bit range [0, %d)", subnet, bits, 1<<bits)
	}
	ipv4[3] = 0
	ipv4[2] = ipv4[2] + subnet*(1<<(24-ipv4[4])) // relies on prefix length in range [16, 24]
	return ipv4, nil
}

func (ipv4 *IPv4) UnmarshalJSON(b []byte) (err error) {
	return parseIPv4(strings.Trim(string(b), `"`), ipv4)
}

func parseIPv4(s string, ipv4 *IPv4) (err error) {
	fields := strings.FieldsFunc(
		s,
		func(r rune) bool { return r == '.' || r == '/' },
	)
	if len(fields) != len(ipv4) {
		return fmt.Errorf("malformed IPv4 %s", s)
	}
	for i := 0; i < len(ipv4); i++ {
		if ipv4[i], err = strconv.Atoi(fields[i]); err != nil {
			return
		}
	}
	return
}
