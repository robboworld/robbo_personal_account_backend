//go:build !unix

package blobstore

func (s *LocalFS) FreeBytes() (int64, error) { return -1, nil }

func (s *LocalFS) TotalBytes() (int64, error) { return -1, nil }
