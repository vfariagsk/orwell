package domain

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

// IPAddress represents an IPv4 address
type IPAddress struct {
	Address string
}

// NewIPAddress creates a new IP address from a string
func NewIPAddress(address string) (*IPAddress, error) {
	if !isValidIPv4(address) {
		return nil, fmt.Errorf("invalid IPv4 address: %s", address)
	}
	return &IPAddress{Address: address}, nil
}

// String returns the string representation of the IP address
func (ip *IPAddress) String() string {
	return ip.Address
}

// isValidIPv4 checks if the given string is a valid IPv4 address
func isValidIPv4(address string) bool {
	parsedIP := net.ParseIP(address)
	return parsedIP != nil && parsedIP.To4() != nil
}

// IPGenerator defines the interface for generating IP addresses
type IPGenerator interface {
	GenerateIPs(count int) ([]*IPAddress, error)
	GenerateRandomIPs(count int) ([]*IPAddress, error)
	GenerateSequentialIPs(startIP string, count int) ([]*IPAddress, error)
}

// IPGeneratorService implements the IP generation logic with permutation-based randomization
type IPGeneratorService struct {
	rand        *rand.Rand
	permutation *IPPermutation
}

// IPPermutation implements a bijective permutation for 32-bit IP space
type IPPermutation struct {
	seed uint32
}

// NewIPPermutation creates a new IP permutation generator
func NewIPPermutation(seed uint32) *IPPermutation {
	return &IPPermutation{seed: seed}
}

// permute32 implements a bijective permutation function for 32-bit integers
// Using a combination of multiplication and XOR operations
func (p *IPPermutation) permute32(x uint32) uint32 {
	// Multiplicative inverse for 32-bit numbers
	// Using a prime number close to 2^32
	const prime = 0x7fffffff // 2^31 - 1 (Mersenne prime)

	// Apply permutation using multiplication and XOR
	x = (x * prime) ^ p.seed
	x = x ^ (x >> 16)
	x = x * 0x85ebca6b
	x = x ^ (x >> 13)
	x = x * 0xc2b2ae35
	x = x ^ (x >> 16)

	return x
}

// isValidPublicIP checks if an IP address is valid for public use
// Excludes private ranges and special purpose addresses
func isValidPublicIP(ip net.IP) bool {
	if ip == nil || ip.To4() == nil {
		return false
	}

	ip4 := ip.To4()

	// Convert to uint32 for easier range checking
	ipUint := uint32(ip4[0])<<24 + uint32(ip4[1])<<16 + uint32(ip4[2])<<8 + uint32(ip4[3])

	// Exclude ranges:
	// 0.0.0.0/8 (0.0.0.0 - 0.255.255.255)
	if ipUint <= 0x00FFFFFF {
		return false
	}

	// 10.0.0.0/8 (10.0.0.0 - 10.255.255.255)
	if ipUint >= 0x0A000000 && ipUint <= 0x0AFFFFFF {
		return false
	}

	// 127.0.0.0/8 (127.0.0.0 - 127.255.255.255)
	if ipUint >= 0x7F000000 && ipUint <= 0x7FFFFFFF {
		return false
	}

	// 192.168.0.0/16 (192.168.0.0 - 192.168.255.255)
	if ipUint >= 0xC0A80000 && ipUint <= 0xC0A8FFFF {
		return false
	}

	// 224.0.0.0/4 (224.0.0.0 - 239.255.255.255) - Multicast
	if ipUint >= 0xE0000000 && ipUint <= 0xEFFFFFFF {
		return false
	}

	// 240.0.0.0/4 (240.0.0.0 - 255.255.255.255) - Reserved
	if ipUint >= 0xF0000000 {
		return false
	}

	return true
}

// NewIPGeneratorService creates a new IP generator service
func NewIPGeneratorService() *IPGeneratorService {
	seed := uint32(time.Now().UnixNano())
	return &IPGeneratorService{
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		permutation: NewIPPermutation(seed),
	}
}

// GenerateIPs generates the specified number of IP addresses
func (s *IPGeneratorService) GenerateIPs(count int) ([]*IPAddress, error) {
	return s.GenerateRandomIPs(count)
}

// GenerateRandomIPs generates random IPv4 addresses using permutation-based randomization
func (s *IPGeneratorService) GenerateRandomIPs(count int) ([]*IPAddress, error) {
	var ips []*IPAddress
	generated := make(map[uint32]bool)

	attempts := 0
	maxAttempts := count * 100 // Prevent infinite loops

	for len(ips) < count && attempts < maxAttempts {
		attempts++

		// Generate a random 32-bit number
		randomUint := uint32(s.rand.Int31())

		// Apply permutation to get a pseudo-random IP
		permutedIP := s.permutation.permute32(randomUint)

		// Skip if already generated
		if generated[permutedIP] {
			continue
		}

		// Convert to IP address
		ipBytes := []byte{
			byte(permutedIP >> 24),
			byte(permutedIP >> 16),
			byte(permutedIP >> 8),
			byte(permutedIP),
		}

		ip := net.IP(ipBytes)

		// Check if it's a valid public IP
		if !isValidPublicIP(ip) {
			continue
		}

		// Mark as generated
		generated[permutedIP] = true

		// Create IP address object
		ipAddress, err := NewIPAddress(ip.String())
		if err != nil {
			continue
		}

		ips = append(ips, ipAddress)
	}

	if len(ips) < count {
		return nil, fmt.Errorf("could only generate %d valid IPs out of %d requested", len(ips), count)
	}

	return ips, nil
}

// GenerateSequentialIPs generates sequential IPv4 addresses starting from the given IP
func (s *IPGeneratorService) GenerateSequentialIPs(startIP string, count int) ([]*IPAddress, error) {
	parsedIP := net.ParseIP(startIP)
	if parsedIP == nil || parsedIP.To4() == nil {
		return nil, fmt.Errorf("invalid starting IP address: %s", startIP)
	}

	var ips []*IPAddress
	ip := parsedIP.To4()

	for i := 0; i < count; i++ {
		// Skip invalid ranges
		for !isValidPublicIP(ip) {
			// Increment IP address
			for j := 3; j >= 0; j-- {
				ip[j]++
				if ip[j] != 0 {
					break
				}
			}
		}

		ipAddress, err := NewIPAddress(ip.String())
		if err != nil {
			return nil, err
		}

		ips = append(ips, ipAddress)

		// Increment IP address for next iteration
		for j := 3; j >= 0; j-- {
			ip[j]++
			if ip[j] != 0 {
				break
			}
		}
	}

	return ips, nil
}

// incrementIP increments an IP address by 1
func incrementIP(ipStr string) string {
	parsedIP := net.ParseIP(ipStr)
	if parsedIP == nil {
		return ipStr
	}

	ip := parsedIP.To4()
	if ip == nil {
		return ipStr
	}

	// Increment IP address
	for j := 3; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}

	return ip.String()
}
