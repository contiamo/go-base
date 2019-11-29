package router

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"
)

// DefaultPackageName used in the router's source code
const DefaultPackageName = "openapi"

// Options represent all the possible options of the generator
type Options struct {
	// PackageName of the generated router source code (`DefaultPackageName` by default)
	PackageName string

	// FailNoGroup if true the generator returns an error if an endpoint without
	// `x-handler-group` attribute was found. Otherwise, this endpoint will be skipped silently.
	FailNoGroup bool

	// FailNoOperationID if true the generator returns an error if an endpoint without
	// `operationId` attribute was found. Otherwise, this endpoint will be skipped silently.
	FailNoOperationID bool
}

// Generate writes a chi router source code into `router` reading the YAML definition of the
// Open API 3.0 spec from the `specFile`. It supports options `opts` (see `Options`).
//
// All included into generation endpoints must have `operationId` and `x-handler-group`
// attributes. Depending on the `opts` generator will either produce an error or skip
// endpoints without these attributes.
func Generate(specFile io.Reader, router io.Writer, opts Options) (err error) {
	decoder := yaml.NewDecoder(specFile)
	spec := spec{}
	err = decoder.Decode(&spec)
	if err != nil {
		return err
	}

	if opts.PackageName == "" {
		opts.PackageName = DefaultPackageName
	}

	tctx, err := createTemplateCtx(spec, opts)
	if err != nil {
		return err
	}

	return routerTemplate.Execute(router, tctx)
}

type templateCtx struct {
	PackageName   string
	Spec          spec
	Groups        map[string]*handlerGroup
	PathsByGroups map[string]*pathsInGroup
}

type pathsInGroup struct {
	AllowedMethodsByPaths map[string]*methodsInPath
}

type methodsInPath struct {
	OperationsByMethods map[string]string
}

type handlerGroup struct {
	Endpoints []endpoint
}

type endpoint struct {
	Summary     string `yaml:"summary"`
	Description string `yaml:"description"`
	OperationID string `yaml:"operationId"`
	Group       string `yaml:"x-handler-group"`
}

type path struct {
	GET    *endpoint `yaml:"get"`
	POST   *endpoint `yaml:"post"`
	PUT    *endpoint `yaml:"put"`
	PATCH  *endpoint `yaml:"patch"`
	DELETE *endpoint `yaml:"delete"`
}

type info struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

type spec struct {
	Info  info            `yaml:"info"`
	Paths map[string]path `yaml:"paths"`
}

func createTemplateCtx(spec spec, opts Options) (out templateCtx, err error) {
	out.PackageName = opts.PackageName
	out.Spec = spec
	out.Groups = make(map[string]*handlerGroup)
	out.PathsByGroups = make(map[string]*pathsInGroup)

	for path, definition := range spec.Paths {
		err = setEndpoint(&out, opts, http.MethodGet, path, definition.GET)
		if err != nil {
			return out, err
		}

		err = setEndpoint(&out, opts, http.MethodPost, path, definition.POST)
		if err != nil {
			return out, err
		}

		err = setEndpoint(&out, opts, http.MethodPut, path, definition.PUT)
		if err != nil {
			return out, err
		}

		err = setEndpoint(&out, opts, http.MethodPatch, path, definition.PATCH)
		if err != nil {
			return out, err
		}

		err = setEndpoint(&out, opts, http.MethodDelete, path, definition.DELETE)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func setEndpoint(out *templateCtx, opts Options, method, path string, e *endpoint) error {
	if e == nil {
		return nil
	}
	if e.Group == "" {
		if opts.FailNoGroup {
			return fmt.Errorf("`%s %s` does not have the `x-handler-group` value", method, path)
		}
		return nil
	}
	if e.OperationID == "" {
		if opts.FailNoOperationID {
			return fmt.Errorf("`%s %s` does not have the `operationId` value", method, path)
		}
		return nil
	}

	group := out.Groups[e.Group]
	if group == nil {
		group = &handlerGroup{}
		out.Groups[e.Group] = group
	}
	group.Endpoints = append(group.Endpoints, *e)

	exPathsInGroup := out.PathsByGroups[e.Group]
	if exPathsInGroup == nil {
		exPathsInGroup = &pathsInGroup{
			AllowedMethodsByPaths: make(map[string]*methodsInPath),
		}
		out.PathsByGroups[e.Group] = exPathsInGroup
	}

	exMethodsInPath := exPathsInGroup.AllowedMethodsByPaths[path]
	if exMethodsInPath == nil {
		exMethodsInPath = &methodsInPath{
			OperationsByMethods: make(map[string]string, 5), // we have only 5 HTTP methods
		}
		exPathsInGroup.AllowedMethodsByPaths[path] = exMethodsInPath
	}

	exMethodsInPath.OperationsByMethods[method] = e.OperationID

	return nil
}

func httpMethod(str string) string {
	return firstUpper(strings.ToLower(str))
}
func firstLower(str string) string {
	return strings.ToLower(str[0:1]) + str[1:]
}
func firstUpper(str string) string {
	return strings.ToUpper(str[0:1]) + str[1:]
}
func commentBlock(str string) string {
	return "// " + strings.Replace(strings.TrimSpace(str), "\n", "\n// ", -1)
}

// template definitions
var (
	fmap = template.FuncMap{
		"firstLower":   firstLower,
		"firstUpper":   firstUpper,
		"httpMethod":   httpMethod,
		"commentBlock": commentBlock,
	}
	routerTemplateSource = `package {{ .PackageName }}

// This file is auto-generated, don't modify it manually

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
)

{{range $name, $group := .Groups }}
// {{ $name }}Handler handles the operations of the '{{ $name }}' handler group.
type {{ $name }}Handler interface {
{{- range $idx, $e := $group.Endpoints }}
	{{ (printf "%s %s" $e.OperationID $e.Description) | commentBlock }}
	{{ $e.OperationID }}(w http.ResponseWriter, r *http.Request)
{{- end}}
}
{{end}}
// NewRouter creates a new router for the spec and the given handlers.
// {{ .Spec.Info.Title }}
//
{{ .Spec.Info.Description | commentBlock }}
//
// {{ .Spec.Info.Version }}
//
func NewRouter(
{{- range $group, $def := .PathsByGroups }}
	{{ $group | firstLower}}Handler {{ $group | firstUpper }}Handler,
{{- end}}
) http.Handler {

	r := chi.NewRouter()
{{range $group, $pathsInGroup := .PathsByGroups }}
// '{{ $group }}' group
{{ range $path, $methodsInPath := $pathsInGroup.AllowedMethodsByPaths }}
// '{{ $path }}'
r.Options("{{ $path }}", optionsHandlerFunc(
{{- range $method, $operation := $methodsInPath.OperationsByMethods }}
	http.Method{{ $method | httpMethod }},
{{- end}}
))

{{- range $method, $operation := $methodsInPath.OperationsByMethods }}
r.{{ $method | httpMethod }}("{{ $path }}", {{ $group | firstLower }}Handler.{{ $operation }})
{{- end}}
{{end}}
{{- end}}
	return r
}

func optionsHandlerFunc(allowedMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	}
}
`
	routerTemplate = template.Must(
		template.New("router").
			Funcs(fmap).
			Parse(routerTemplateSource),
	)
)
