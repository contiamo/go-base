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

func goTypeFromSpec(schemaRef *openapi3.SchemaRef) string {
	schema := schemaRef.Value
	propertyType := schemaRef.Value.Type
	switch propertyType {
	case "object":
		if schemaRef.Ref != "" {
			propertyType = filepath.Base(schemaRef.Ref)
		} else {
			subType := "interface{}"
			if schema.AdditionalProperties != nil {
				subType = goTypeFromSpec(schema.AdditionalProperties)
			}
			propertyType = "map[string]" + subType
		}
	case "string":
		if schema.Format == "date-time" || schema.Format == "time" {
			propertyType = "time.Time"
		}
		if len(schema.Enum) > 0 && schemaRef.Ref != "" {
			propertyType = filepath.Base(schemaRef.Ref)
		}
	case "array":
		propertyType = "[]" + goTypeFromSpec(schema.Items)
	case "boolean":
		propertyType = "bool"
	case "integer":
		propertyType = "int32"
	case "number":
		propertyType = "float32"
	case "":
		propertyType = "interface{}"
	}
	if schema.Nullable && !strings.HasPrefix(propertyType, "[]") && !strings.HasPrefix(propertyType, "map[") {
		propertyType = "*" + propertyType
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
			propertyType := goTypeFromSpec(propSpec)
			if propertyType == "time.Time" || propertyType == "*time.Time" {
				found := false
				for _, i := range modelContext.Imports {
					if i == "time" {
						found = true
						break
					}
				}
				if !found {
					modelContext.Imports = append(modelContext.Imports, "time")
				}
			}
			omitEmpty := true
			for _, required := range s.Value.Required {
				if required == propName {
					omitEmpty = false
					break
				}
			}
			jsonTags := "`json:\"" + propName
			if omitEmpty {
				jsonTags += ",omitempty"
			}
			jsonTags += "\"`"
			modelContext.Properties = append(modelContext.Properties, propertyContext{
				Name:     tpl.ToPascalCase(propName),
				Type:     propertyType,
				JSONTags: jsonTags,
			})
		}
		sort.Sort(modelContext.Properties)
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
	Imports     []string
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

import (
{{- range .Imports}}
	"{{.}}"
{{- end}}
)

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
