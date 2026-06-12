//go:build windows

package sing_tun

import (
	"net"
	"unsafe"

	"github.com/metacubex/mihomo/log"

	"golang.org/x/sys/windows"
)

var (
	modIphlpapi              = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetIpForwardTable2   = modIphlpapi.NewProc("GetIpForwardTable2")
	procDeleteIpForwardEntry2 = modIphlpapi.NewProc("DeleteIpForwardEntry2")
)

type addressFamily uint16

const (
	AF_INET  addressFamily = 2
	AF_INET6 addressFamily = 23
)

type netLUID uint64

type sockAddrInet struct {
	Family uint16
	// Padding to match SOCKADDR_INET union size
	data [126]byte
}

type mibIPforwardRow2 struct {
	InterfaceLUID    netLUID
	InterfaceIndex   uint32
	DestinationPrefix sockAddrInet
	NextHop          sockAddrInet
	SitePrefixLength uint8
	ValidLifetime    uint32
	PreferredLifetime uint32
	Metric           uint32
	Protocol         uint32
	Loopback         uint8
	Autoconfigure    uint8
	Publish          uint8
	Immortal         uint8
	Age              uint32
	Origin           uint32
}

type mibIPforwardTable2 struct {
	NumberOfEntries uint32
	Table           [1]mibIPforwardRow2
}

// cleanupRoutesForInterface removes all routes associated with the given interface index.
// This is a workaround for sing-tun not cleaning up routes on Close().
func cleanupRoutesForInterface(ifaceName string) error {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil // Interface not found, nothing to clean up
	}

	// Clean up IPv4 routes
	if err := cleanupRoutesForFamily(AF_INET, uint32(iface.Index)); err != nil {
		log.Warnln("[TUN] Failed to cleanup IPv4 routes: %s", err)
	}

	// Clean up IPv6 routes
	if err := cleanupRoutesForFamily(AF_INET6, uint32(iface.Index)); err != nil {
		log.Warnln("[TUN] Failed to cleanup IPv6 routes: %s", err)
	}

	return nil
}

func cleanupRoutesForFamily(family addressFamily, ifaceIndex uint32) error {
	var tab *mibIPforwardTable2

	// First call to get the required buffer size
	ret, _, _ := procGetIpForwardTable2.Call(
		uintptr(family),
		uintptr(unsafe.Pointer(&tab)),
	)
	if ret != 0 {
		// Try with a pre-allocated buffer
		buf := make([]byte, 8*1024) // 8KB should be enough
		tab = (*mibIPforwardTable2)(unsafe.Pointer(&buf[0]))
		ret, _, _ = procGetIpForwardTable2.Call(
			uintptr(family),
			uintptr(unsafe.Pointer(&tab)),
		)
		if ret != 0 {
			return nil // Failed to get route table, skip cleanup
		}
	}

	if tab == nil || tab.NumberOfEntries == 0 {
		return nil
	}

	// Iterate through the route table and delete routes for our interface
	tablePtr := uintptr(unsafe.Pointer(&tab.Table[0]))
	rowSize := uintptr(unsafe.Sizeof(tab.Table[0]))

	for i := uint32(0); i < tab.NumberOfEntries; i++ {
		row := (*mibIPforwardRow2)(unsafe.Pointer(tablePtr + rowSize*uintptr(i)))
		if row.InterfaceIndex == ifaceIndex {
			procDeleteIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
		}
	}

	return nil
}

// getInterfaceLUID returns the LUID for the given interface name.
// This is used internally for route management.
func getInterfaceLUID(ifaceName string) (netLUID, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return 0, err
	}

	// Convert interface index to LUID using Windows API
	var luid netLUID
	modIphlpapi.NewProc("ConvertInterfaceIndexToLuid").Call(
		uintptr(iface.Index),
		uintptr(unsafe.Pointer(&luid)),
	)

	return luid, nil
}
