package models

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var accessorsSpec = `
openapi: 3.0.0
info:
  version: 0.1.0
  title: Hub Service
components:
    schemas:
        TestType:
            type: object
            properties:
              foo:
                type: int
              bar:
                type: string
              baz:
                $ref: "#/components/schemas/SubType"
        SubType:
            type: object
            properties:
                foo:
                    type: string
`

func TestGenerateAccessors(t *testing.T) {

	dname, err := ioutil.TempDir("", "accessordir")
	require.NoError(t, err)

	defer os.RemoveAll(dname)

	opts := Options{PackageName: "testpkg"}

	specReader := strings.NewReader(accessorsSpec)
	err = GenerateAccessors(specReader, dname, opts)
	require.NoError(t, err)

	content, err := ioutil.ReadFile(filepath.Join(dname, "accessors.go"))
	require.NoError(t, err)
	fmt.Println(string(content))
	expectedFilterType, err := ioutil.ReadFile("testdata/accessors.go")
	require.NoError(t, err)
	expectedFilterType = bytes.TrimSpace(expectedFilterType)
	content = bytes.TrimSpace(content)
	require.Equal(t, string(expectedFilterType), string(content))
}
