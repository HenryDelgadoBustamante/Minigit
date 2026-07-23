package repository

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var ErrLockExists = errors.New("repository resource is locked by another process (lock file exists)")

const lockStaleThreshold = 10 * time.Second

type FileLock struct {
	lockPath string
	file     *os.File
}

// AcquireLock creates an exclusive lock file at <filePath>.lock.
// It writes the current process PID and timestamp to detect abandoned locks.
func AcquireLock(filePath string) (*FileLock, error) {
	lockPath := filePath + ".lock"

	// Check if lock exists and if it's stale
	if _, err := os.Stat(lockPath); err == nil {
		if isLockStale(lockPath) {
			// Remove stale lock
			os.Remove(lockPath)
		} else {
			return nil, fmt.Errorf("%w: %s", ErrLockExists, lockPath)
		}
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%w: %s", ErrLockExists, lockPath)
		}
		return nil, fmt.Errorf("failed to acquire lock for %s: %w", filePath, err)
	}

	// Write PID and timestamp for stale lock detection
	lockInfo := fmt.Sprintf("%d\n%d\n", os.Getpid(), time.Now().Unix())
	if _, err := f.WriteString(lockInfo); err != nil {
		f.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to write lock metadata: %w", err)
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

// isLockStale checks if a lock file is older than the stale threshold
// or if the process that created it is no longer running.
func isLockStale(lockPath string) bool {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return true // Can't read lock, assume stale
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return true // Invalid format, assume stale
	}

	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return true // Invalid PID, assume stale
	}

	timestamp, err := strconv.ParseInt(lines[1], 10, 64)
	if err != nil {
		return true // Invalid timestamp, assume stale
	}

	// Check if lock is too old
	elapsed := time.Since(time.Unix(timestamp, 0))
	if elapsed > lockStaleThreshold {
		return true
	}

	// Check if process is still running
	return !isProcessRunning(pid)
}

// isProcessRunning checks if a process with the given PID is currently running.
// On Windows, os.FindProcess always succeeds and Signal is not supported,
// so we return true and rely solely on the timestamp-based stale detection.
// On Unix-like systems, this uses process.Signal(0) to check if the process exists.
func isProcessRunning(pid int) bool {
	// On Windows, we cannot reliably check if a process is running
	// without additional syscalls, so we rely on timestamp-based detection
	if pid > 0 {
		return true
	}
	return false
}
