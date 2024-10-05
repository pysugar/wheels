//go:build !windows
// +build !windows

package internet

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

// Acquire lock
func (fl *FileLocker) Acquire() error {
	f, err := os.Create(fl.path)
	if err != nil {
		return err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return fmt.Errorf("failed to lock file: %s, err: %v", fl.path, err)
	}
	fl.file = f
	return nil
}

// Release lock
func (fl *FileLocker) Release() {
	if err := unix.Flock(int(fl.file.Fd()), unix.LOCK_UN); err != nil {
		log.Printf("failed to unlock file: %s, err: %v", fl.path, err)
	}
	if err := fl.file.Close(); err != nil {
		log.Printf("failed to close file: %s, err: %v", fl.path, err)
	}
	if err := os.Remove(fl.path); err != nil {
		log.Printf("failed to remove file: %s, err: %v", fl.path, err)
	}
}
