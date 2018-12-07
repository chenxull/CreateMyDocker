package network

import (
	"net"
	"testing"
)

func TestAllocate(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.0.0/24")
	for i := 1; i < 100; i++ {
		ip, _ := ipAllocator.Allocate(ipnet)
		t.Logf("alloc ip : %v", ip)
	}

}

func TestRelease(t *testing.T) {
	ip, ipnet, _ := net.ParseCIDR("192.168.0.2/24")
	ipAllocator.Release(ipnet, &ip)
}
