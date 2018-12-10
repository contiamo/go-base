package fileutils

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// "recursively copy a file object, info must be non-nil
func Copy(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return dcopy(src, dest, info)
	}
	return fcopy(src, dest, info)
}

// fcopy will copy a file with the same mode as the src file
func fcopy(src, dest string, srcStat os.FileInfo) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = os.Chmod(f.Name(), srcStat.Mode()); err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	return err
}

// dcopy will recursively copy a directory to dest
func dcopy(src, dest string, srcStat os.FileInfo) error {
	if err := os.MkdirAll(dest, srcStat.Mode()); err != nil {
		return err
	}

	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if err := Copy(
			filepath.Join(src, info.Name()),
			filepath.Join(dest, info.Name()),
		); err != nil {
			return err
		}
	}

	return nil
}
