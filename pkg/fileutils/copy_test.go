package fileutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tmpDir(t *testing.T) (path string, cleanup func()) {
	path, err := os.MkdirTemp("/tmp", "go-base")
	require.NoError(t, err)
	return path, func() {
		err = os.RemoveAll(path)
		require.NoError(t, err)
	}
}

func TestSimpleCopy(t *testing.T) {
	tmp, cleanup := tmpDir(t)
	defer cleanup()

	src := "testdata/test.txt"
	dst := filepath.Join(tmp, "file")

	err := Copy(src, dst)
	assert.NoError(t, err)

	srcBs, _ := os.ReadFile(src)
	destBs, _ := os.ReadFile(dst)
	assert.EqualValues(t, srcBs, destBs)
}

func TestCopyFolderWithFolderRename(t *testing.T) {
	src, cleanup1 := tmpDir(t)
	defer cleanup1()

	dst, cleanup2 := tmpDir(t)
	defer cleanup2()

	f1 := filepath.Join(src, "f1")
	//nolint:gosec  // writing test file is ok
	err := os.WriteFile(f1, []byte("foobar"), 0644)
	assert.NoError(t, err)

	err = Copy(src, dst)
	assert.NoError(t, err)

	f1Bs, _ := os.ReadFile(filepath.Join(dst, "f1"))
	assert.EqualValues(t, f1Bs, []byte("foobar"))
}

func TestCopyNonExistentFile(t *testing.T) {
	src := "doesnt-exist"
	dst, cleanup := tmpDir(t)
	defer cleanup()

	err := Copy(src, dst)
	assert.Error(t, err)
}

func TestCopyNonReadableFile(t *testing.T) {
	src, cleanup1 := tmpDir(t)
	defer cleanup1()

	dst, cleanup2 := tmpDir(t)
	defer cleanup2()

	err := os.WriteFile(filepath.Join(src, "f1"), []byte{}, 0)
	assert.NoError(t, err)

	err = Copy(src, dst)
	assert.Error(t, err)
}

func TestCopyNonReadableDirectory(t *testing.T) {
	tmp, cleanup := tmpDir(t)
	defer cleanup()

	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	err := os.MkdirAll(src, 0)
	assert.NoError(t, err)

	err = Copy(src, dst)
	assert.Error(t, err)
}
