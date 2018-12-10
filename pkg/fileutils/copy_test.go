package fileutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleCopy(t *testing.T) {
	src := "testdata/test.txt"
	dest := "/tmp/test.txt"
	err := Copy(src, dest)
	assert.NoError(t, err)
	srcBs, _ := ioutil.ReadFile(src)
	destBs, _ := ioutil.ReadFile(dest)
	assert.EqualValues(t, srcBs, destBs)
}

func TestCopyFolderWithFolderRename(t *testing.T) {
	src := "/tmp/source-dir"
	os.MkdirAll(src, 0755)
	f1 := filepath.Join(src, "f1")
	ioutil.WriteFile(f1, []byte("foobar"), 0644)
	dest := "/tmp/dest-dir"
	err := Copy(src, dest)
	assert.NoError(t, err)
	f1Bs, _ := ioutil.ReadFile(filepath.Join(dest, "f1"))
	assert.EqualValues(t, f1Bs, []byte("foobar"))
}

func TestCopyNonExistentFile(t *testing.T) {
	src := "doesnt-exist"
	dest := "/tmp/target"
	err := Copy(src, dest)
	assert.Error(t, err)
}

func TestCopyNonReadableFile(t *testing.T) {
	os.MkdirAll("/tmp/src", 0755)
	ioutil.WriteFile("/tmp/src/f1", []byte{}, 0)
	src := "/tmp/src"
	dest := "/tmp/dest"
	err := Copy(src, dest)
	assert.Error(t, err)
}

func TestCopyNonReadableDirectory(t *testing.T) {
	os.MkdirAll("/tmp/src", 0)
	src := "/tmp/src"
	dest := "/tmp/dest"
	err := Copy(src, dest)
	assert.Error(t, err)
}
