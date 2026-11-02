//go:build unix

package blobstore

import "syscall"

func (s *LocalFS) statfs() (syscall.Statfs_t, error) {
	var st syscall.Statfs_t
	err := syscall.Statfs(s.root, &st)
	return st, err
}

func (s *LocalFS) FreeBytes() (int64, error) {
	st, err := s.statfs()
	if err != nil {
		return 0, err
	}
	return int64(uint64(st.Bavail) * uint64(st.Bsize)), nil
}

// TotalBytes is the size of the file system under the store.
func (s *LocalFS) TotalBytes() (int64, error) {
	st, err := s.statfs()
	if err != nil {
		return 0, err
	}
	return int64(uint64(st.Blocks) * uint64(st.Bsize)), nil
}
