//go:build !darwin && !windows

package credentials

func newPlatformKeyringStore() Store {
	return nil
}
