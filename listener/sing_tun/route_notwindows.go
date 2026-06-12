//go:build !windows

package sing_tun

// cleanupRoutesForInterface is a no-op on non-Windows platforms.
func cleanupRoutesForInterface(ifaceName string) error {
	return nil
}
