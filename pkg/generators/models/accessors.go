package models

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/sirupsen/logrus"

	tpl "github.com/contiamo/go-base/v2/pkg/generators/templates"
)

// GenerateAccessors outputs accessors for all the models
func GenerateAccessors(specFile io.Reader, dst string, opts Options) error {
	if opts.PackageName == "" {
		opts.PackageName = DefaultPackageName
	}

	data, err := ioutil.ReadAll(specFile)
	if err != nil {
		return fmt.Errorf("can not read spec file: %w", err)
	}
	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(data)
	if err != nil {
		return fmt.Errorf("can not parse the OpenAPI spec: %w", err)
	}

	templateCtx := accessorTemplateCtx{
		SpecTitle:   swagger.Info.Title,
		SpecVersion: swagger.Info.Version,
		PackageName: opts.PackageName,
	}

	// to do sort and iterate over the sorted schema
	for modelName, modelSpec := range swagger.Components.Schemas {
		if modelSpec.Value.Type == "object" {
			for propName, propSpec := range modelSpec.Value.Properties {
				propertyType := goTypeFromSpec(propSpec.Ref, propSpec.Value)
				templateCtx.Getters = append(templateCtx.Getters, getterTemplateCtx{
					ModelName:  modelName,
					FieldName:  tpl.ToPascalCase(propName),
					ReturnType: propertyType,
				})
				templateCtx.Setters = append(templateCtx.Setters, setterTemplateCtx{
					ModelName:    modelName,
					FieldName:    tpl.ToPascalCase(propName),
					ArgumentType: propertyType,
				})
			}
		}
	}

	filename := filepath.Join(dst, "accessors.go")
	logrus.Debugf("writings %s\n", filename)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	err = accessorTemplate.Execute(f, templateCtx)
	if err != nil {
		return fmt.Errorf("failed to render output file: %w", err)
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}
	return nil
}

type accessorTemplateCtx struct {
	SpecTitle   string
	SpecVersion string
	PackageName string
	Getters     []getterTemplateCtx
	Setters     []setterTemplateCtx
}

type getterTemplateCtx struct {
	ModelName  string
	FieldName  string
	ReturnType string
}
type setterTemplateCtx struct {
	ModelName    string
	FieldName    string
	ArgumentType string
}

var accessorTemplateSource = `// This file is auto-generated, DO NOT EDIT.
//
// Source:
//     Title: {{.SpecTitle}}
//     Version: {{.SpecVersion}}
package {{ .PackageName }}
{{ range .Getters }}
// Get{{.FieldName}} returns the {{.FieldName}} property
func (m {{.ModelName}}) Get{{.FieldName}}() {{.ReturnType}} {
	return m.{{.FieldName}}
}
{{ end }}{{ range .Setters }}
// Set{{.FieldName}} sets the {{.FieldName}} property
func (m {{.ModelName}}) Set{{.FieldName}}(val {{.ArgumentType}}) {
	m.{{.FieldName}} = val
}
{{ end }}
`
var accessorTemplate = template.Must(template.New("accessor").Parse(accessorTemplateSource))
