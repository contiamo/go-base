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

	tpl "github.com/contiamo/go-base/v2/pkg/generators/templates"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/sirupsen/logrus"
)

type StructCtx struct {
	Filename    string
	SpecTitle   string
	SpecVersion string
	PackageName string
	Name        string
	VarName     string
	Type        string
	Description string
	Fields      FieldCtxs
}

type StructCtxs []StructCtx

func (ctx StructCtxs) Len() int           { return len(ctx) }
func (ctx StructCtxs) Swap(i, j int)      { ctx[i], ctx[j] = ctx[j], ctx[i] }
func (ctx StructCtxs) Less(i, j int) bool { return ctx[i].Name < ctx[j].Name }

type FieldCtx struct {
	Name     string
	VarName  string
	Type     string
	Nullable bool
	Required bool
}

type FieldCtxs []FieldCtx

func (ctx FieldCtxs) Len() int           { return len(ctx) }
func (ctx FieldCtxs) Swap(i, j int)      { ctx[i], ctx[j] = ctx[j], ctx[i] }
func (ctx FieldCtxs) Less(i, j int) bool { return ctx[i].Name < ctx[j].Name }

func GenerateStructs(specFile io.Reader, dst string, opts Options) error {
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

	structs := StructCtxs{}
	// to do sort and iterate over the sorted schema
	for name, s := range swagger.Components.Schemas {
		if len(s.Value.Enum) != 0 {
			fmt.Printf("skipping %s\n", name)
			continue
		}
		fmt.Printf("parsing %s\n", name)

		requiredFields := map[string]bool{}
		for _, fieldName := range s.Value.Required {
			requiredFields[fieldName] = true
		}

		tctx := StructCtx{
			SpecTitle:   swagger.Info.Title,
			SpecVersion: swagger.Info.Version,
			PackageName: opts.PackageName,
			Name:        name,
			VarName:     tpl.ToPascalCase(name),
			Description: s.Value.Description,
			Type:        s.Value.Type,
			Fields:      FieldCtxs{},
		}


		// [x] Handle basic type
		// [x] Handle ref to another schema
		// [ ] Handle arrays
		// [ ] Handle nested types/arrays/structs <- should extract and make recursive
		for name, spec := range s.Value.Properties {
			// handle references to other schemas
			if spec.Ref != "" {
				// we assume tha tthe referenced type has been or will be generated
				fieldType := tpl.ToPascalCase(strings.TrimPrefix(spec.Ref, "#/components/schemas/"))
				field := FieldCtx{
					Name: name,
					VarName: tpl.ToPascalCase(name),
					Type: fieldType,
					Required: requiredFields[name],
				}
				tctx.Fields = append(tctx.Fields, field)
				continue
			}

			// handle actual values
			fieldType := spec.Value.Type
			if spec.Ref == "" && spec.Value.Nullable {
				fieldType = "*"+fieldType
			}


			field := FieldCtx{
				Name: name,
				VarName: tpl.ToPascalCase(name),
				Type: fieldType,
				Nullable: spec.Value.Nullable,
				Required: requiredFields[name],
			}
			tctx.Fields = append(tctx.Fields, field)
		}

		sort.Sort(tctx.Fields)
		structs = append(structs, tctx)
	}

	sort.Sort(structs)

	for _, tctx := range structs {
		modelName := strings.ToLower(strings.ReplaceAll(fmt.Sprintf("model_%s.go", tpl.ToSnakeCase(tctx.Name)), " ", "_"))
		filename := filepath.Join(dst, modelName)
		logrus.Debugf("writing %s\n", filename)
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		if len(tctx.Fields) == 0 {
			err = interfaceTemplate.Execute(f, tctx)
		} else {
			err = structTemplate.Execute(f, tctx)
		}
		if err != nil {
			return fmt.Errorf("failed to generate model code: %w", err)
		}

		err = f.Close()
		if err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}
	}

	return nil
}


var interfaceTemplate = template.Must(
	template.New("interfaceModel").
		Funcs(fmap).
		Parse(InterfaceTemplateSource),
)

var structTemplate = template.Must(
	template.New("structModel").
		Funcs(fmap).
		Parse(StructTemplateSource),
)

// StructTemplateSource is the template used to generate struct models
var StructTemplateSource = `
// This file is auto-generated, DO NOT EDIT.
//
// Source:
//     Title: {{.SpecTitle}}
//     Version: {{.SpecVersion}}
package {{ .PackageName }}

{{ (printf "%s %s" .VarName .Description) | commentBlock }}
type {{.VarName}} struct {
	{{- range $v := .Fields }}
	{{$v.VarName}} {{$v.Type}} {{ (jsonTag $v.Name $v.Required) }}
	{{- end}}
}
`

var InterfaceTemplateSource = `
// This file is auto-generated, DO NOT EDIT.
//
// Source:
//     Title: {{.SpecTitle}}
//     Version: {{.SpecVersion}}
package {{ .PackageName }}

{{ (printf "%s %s" .VarName .Description) | commentBlock }}
type {{.VarName}} interface{}
`
