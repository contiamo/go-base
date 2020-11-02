package models

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGenerateStruct(t *testing.T) {
	dname, err := ioutil.TempDir("", "structdir")
	require.NoError(t, err)

	defer os.RemoveAll(dname)

	logrus.SetLevel(logrus.DebugLevel)

	opts := Options{PackageName: "testpkg"}

	specData, err := ioutil.ReadFile("testdata/struct_test_spec.yaml")
	require.NoError(t, err)

	specReader := bytes.NewReader(specData)
	err = GenerateStructs(specReader, dname, opts)
	require.NoError(t, err)

	content, err := ioutil.ReadFile(filepath.Join(dname, "model_connection_spec.go"))
	require.NoError(t, err)

	expectedContent, err := ioutil.ReadFile("testdata/models_connection_spec.go")
	require.NoError(t, err)
	expectedContent = bytes.TrimSpace(expectedContent)
	content = bytes.TrimSpace(content)
	require.Equal(t, string(expectedContent), string(content))


    content, err = ioutil.ReadFile(filepath.Join(dname, "model_user.go"))
	require.NoError(t, err)

	expectedContent, err = ioutil.ReadFile("testdata/model_user.go")
	require.NoError(t, err)
	expectedContent = bytes.TrimSpace(expectedContent)
	content = bytes.TrimSpace(content)
	require.Equal(t, string(expectedContent), string(content))
}

// func ls(name string) error {
// 	fmt.Println(name + ":")
// 	files, err := ioutil.ReadDir(name)
// 	if err != nil {
// 		return err
// 	}

// 	for _, file := range files {
// 		fmt.Println(file.Name())
// 	}

// 	return nil
// }
