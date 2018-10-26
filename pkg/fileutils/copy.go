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
		return dcopy(src, dest)
	}
	return fcopy(src, dest)
}

// fcopy will copy a file with the same mode as the src file
func fcopy(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = os.Chmod(f.Name(), info.Mode()); err != nil {
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
func dcopy(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, info.Mode()); err != nil {
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
