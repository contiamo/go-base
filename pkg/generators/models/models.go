package models

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/sirupsen/logrus"

	tpl "github.com/contiamo/go-base/v2/pkg/generators/templates"
)

func goTypeFromSpec(ref string, spec *openapi3.Schema) string {
	propertyType := spec.Type
	switch propertyType {
	case "object":
		if ref != "" {
			propertyType = filepath.Base(ref)
		} else {
			propertyType = "map[string]interface{}"
		}
	case "string":
		if ref != "" {
			propertyType = filepath.Base(ref)
		}
	case "array":
		propertyType = "[]" + goTypeFromSpec(spec.Items.Ref, spec.Items.Value)
	case "boolean":
		propertyType = "bool"
	case "integer", "number":
		propertyType = "int32"
	case "":
		propertyType = "interface{}"
	}
	return propertyType
}

// GenerateModels outputs the Go enum models with validators
func GenerateModels(specFile io.Reader, dst string, opts Options) error {
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

	models := modelContexts{}

	// to do sort and iterate over the sorted schema
	for name, s := range swagger.Components.Schemas {
		if s.Value.Type != "object" {
			continue
		}

		modelContext := modelContext{
			SpecTitle:   swagger.Info.Title,
			SpecVersion: swagger.Info.Version,
			PackageName: opts.PackageName,
			ModelName:   tpl.ToPascalCase(name),
			Description: s.Value.Description,
		}

		for propName, propSpec := range s.Value.Properties {
			propertyType := goTypeFromSpec(propSpec.Ref, propSpec.Value)
			modelContext.Properties = append(modelContext.Properties, propertyContext{
				Name:     tpl.ToPascalCase(propName),
				Type:     propertyType,
				JSONTags: fmt.Sprintf("`json:\"%s\"`", propName),
			})
		}
		//sort.Sort(modelContext.Properties)
		models = append(models, modelContext)
	}

	sort.Sort(models)
	for _, modelContext := range models {
		modelName := strings.ToLower(strings.ReplaceAll(fmt.Sprintf("model_%s.go", tpl.ToSnakeCase(modelContext.ModelName)), " ", "_"))
		filename := filepath.Join(dst, modelName)

		logrus.Infof("writing %s\n", filename)
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		err = modelTemplate.Execute(f, modelContext)
		err = f.Close()
		if err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}
	}
	return nil
}

type modelContext struct {
	Filename    string
	SpecTitle   string
	SpecVersion string
	PackageName string
	ModelName   string
	Description string
	Properties  propertyContexts
}

type propertyContext struct {
	Name        string
	Description string
	Type        string
	JSONTags    string
}

type modelContexts []modelContext

func (e modelContexts) Len() int           { return len(e) }
func (e modelContexts) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e modelContexts) Less(i, j int) bool { return e[i].ModelName < e[j].ModelName }

type propertyContexts []propertyContext

func (e propertyContexts) Len() int           { return len(e) }
func (e propertyContexts) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e propertyContexts) Less(i, j int) bool { return e[i].Name < e[j].Name }

var modelTemplateSource = `
// This file is auto-generated, DO NOT EDIT.
//
// Source:
//     Title: {{.SpecTitle}}
//     Version: {{.SpecVersion}}
package {{ .PackageName }}

{{ (printf "%s is an object. %s" .ModelName .Description) | commentBlock }}
type {{.ModelName}} struct {
{{- range .Properties}}
	// {{.Name}} {{.Description}}
	{{.Name}} {{.Type}} {{.JSONTags}}
{{- end}}
}

{{- $modelName := .ModelName }}
{{ range .Properties}}
// Get{{.Name}} returns the {{.Name}} property
func (m {{$modelName}})	Get{{.Name}}() {{.Type}} {
	return m.{{.Name}}
}

// Set{{.Name}} sets the {{.Name}} property
func (m {{$modelName}}) Set{{.Name}}(val {{.Type}}) {
	m.{{.Name}} = val
}
{{ end}}
`

var modelTemplate = template.Must(
	template.New("model").
		Funcs(fmap).
		Parse(modelTemplateSource),
)
