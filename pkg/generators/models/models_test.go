package models

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var modelsSpec = `
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
                type: array
                items:
                  $ref: "#/components/schemas/SubType"
        SubType:
            type: object
            properties:
                foo:
                    type: string
`

func TestGenerateModels(t *testing.T) {

	dname, err := ioutil.TempDir("", "modeldir")
	require.NoError(t, err)

	//defer os.RemoveAll(dname)

	opts := Options{PackageName: "testpkg"}

	specReader := strings.NewReader(modelsSpec)
	err = GenerateModels(specReader, dname, opts)
	require.NoError(t, err)

	content, err := ioutil.ReadFile(filepath.Join(dname, "model_test_type.go"))
	require.NoError(t, err)
	fmt.Println(string(content))
	t.Fail()
}
