package dashboard_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// The host lock must actually serialize across open file descriptions:
// flock conflicts bind to the open-file-description pair, so a second
// descriptor on the same file reproduces the cross-process relationship
// exactly (two processes are two descriptions; two fds here are two
// descriptions). Pins the M19 mechanism: held → non-blocking contender
// refuses; released → contender acquires.
func TestBrowserHostLock_SerializesAcrossDescriptions(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(os.TempDir(), browserHostLockFile)

	release := lockBrowserHost(t)

	contender, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("contender open: %v", err)
	}
	defer func() { _ = contender.Close() }()

	if err := syscall.Flock(int(contender.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		_ = syscall.Flock(int(contender.Fd()), syscall.LOCK_UN)
		release()
		t.Fatal("contender acquired the lock while it was held — the host lock does not serialize")
	}

	release()

	if err := syscall.Flock(int(contender.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("contender could not acquire after release: %v", err)
	}
	_ = syscall.Flock(int(contender.Fd()), syscall.LOCK_UN)
}
