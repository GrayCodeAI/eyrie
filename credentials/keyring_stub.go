//go:build !darwin && !linux && !windows

package credentials

func newPlatformKeyringStore() Store {
	return nil
}
