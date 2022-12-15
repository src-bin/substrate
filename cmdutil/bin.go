package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// WritableBinDirname tries to preempt common tar(1) failures we encounter
// when the extraction directory isn't writable for some reason. It returns
// a writable directory name and a nil error or the empty string and a
// non-nil error.
func WritableBinDirname() (string, error) {
	pathname, err := os.Executable()
	if err != nil {
		return "", err
	}
	dirname := filepath.Dir(pathname)
	fi, err := os.Stat(dirname)
	if err != nil {
		return "", err
	}
	perm := fi.Mode().Perm()
	sys := fi.Sys().(*syscall.Stat_t)
	if perm&0200 != 0 && sys.Uid == uint32(os.Geteuid()) {
		// writable by owner, which we are
	} else if perm&0020 != 0 && sys.Gid == uint32(os.Getegid()) {
		// writable by owning group, which is our primary group
		// TODO also check supplemental groups if you want to be fancy
	} else if perm&0002 != 0 {
		// writable by anyone, which is bad but not our problem
	} else {
		return "", fmt.Errorf("%s not writable", dirname)
	}
	return dirname, nil
}
