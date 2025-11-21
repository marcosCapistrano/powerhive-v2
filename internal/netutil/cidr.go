// Package netutil provides network utilities for IP range enumeration and port scanning.
package netutil

import (
	"encoding/binary"
	"fmt"
	"net"
)

// ParseCIDR parses a CIDR notation string and returns all IP addresses in the range.
// Example: "192.168.1.0/24" returns all 254 usable IPs (excludes network and broadcast).
func ParseCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network address and broadcast address for IPv4
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}

	return ips, nil
}

// ParseRange parses an IP range and returns all IPs between start and end (inclusive).
// Example: "192.168.1.1", "192.168.1.10" returns 10 IPs.
func ParseRange(startIP, endIP string) ([]string, error) {
	start := net.ParseIP(startIP)
	if start == nil {
		return nil, fmt.Errorf("invalid start IP: %s", startIP)
	}

	end := net.ParseIP(endIP)
	if end == nil {
		return nil, fmt.Errorf("invalid end IP: %s", endIP)
	}

	start = start.To4()
	end = end.To4()

	if start == nil || end == nil {
		return nil, fmt.Errorf("only IPv4 addresses are supported")
	}

	startInt := ipToUint32(start)
	endInt := ipToUint32(end)

	if startInt > endInt {
		return nil, fmt.Errorf("start IP must be less than or equal to end IP")
	}

	var ips []string
	for i := startInt; i <= endInt; i++ {
		ips = append(ips, uint32ToIP(i).String())
	}

	return ips, nil
}

// IsValidIP checks if the given string is a valid IP address.
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsPrivateIP checks if the IP is in a private range (RFC 1918).
func IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	ip = ip.To4()
	if ip == nil {
		return false
	}

	// 10.0.0.0/8
	if ip[0] == 10 {
		return true
	}

	// 172.16.0.0/12
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}

	// 192.168.0.0/16
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}

	return false
}

// incIP increments an IP address by one.
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// ipToUint32 converts an IPv4 address to a uint32.
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

// uint32ToIP converts a uint32 to an IPv4 address.
func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}
