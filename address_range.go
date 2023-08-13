package sower

import "net/netip"

// AddressRange is ...
type AddressRange []netip.Prefix

// Contains is ...
func (ar AddressRange) Contains(addr netip.Addr) bool {
	for i := range ar {
		if ar[i].Contains(addr) {
			return true
		}
	}
	return false
}
