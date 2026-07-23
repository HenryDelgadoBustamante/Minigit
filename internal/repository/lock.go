package repository

import (
	"errors"
	"fmt"
	"os"
)

var ErrLockExists = errors.New("repository resource is locked by another process (lock file exists)")

type FileLock struct {
	lockPath string
	file     *os.File
}

// AcquireLock creates an exclusive lock file at <filePath>.lock.
func AcquireLock(filePath string) (*FileLock, error) {
	lockPath := filePath + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%w: %s", ErrLockExists, lockPath)
		}
		return nil, fmt.Errorf("failed to acquire lock for %s: %w", filePath, err)
	}

	return &FileLock{
		lockPath: lockPath,
		file:     f,
	}, nil
}

// Unlock releases and removes the lock file.
func (l *FileLock) Unlock() error {
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
	if err := os.Remove(l.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file %s: %w", l.lockPath, err)
	}
	return nil
}
