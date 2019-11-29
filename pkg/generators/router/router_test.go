package router

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHttpMethod(t *testing.T) {
	require.Equal(t, "Get", httpMethod("GET"))
	require.Equal(t, "Get", httpMethod("get"))
	require.Equal(t, "Get", httpMethod("gEt"))
}

func TestFirstLower(t *testing.T) {
	require.Equal(t, "string", firstLower("string"))
	require.Equal(t, "sTRING", firstLower("STRING"))
	require.Equal(t, "string", firstLower("String"))
}

func TestFirstUpper(t *testing.T) {
	require.Equal(t, "String", firstUpper("string"))
	require.Equal(t, "STRING", firstUpper("STRING"))
	require.Equal(t, "String", firstUpper("String"))
}

func TestCommentBlock(t *testing.T) {
	require.Equal(t, "// some\n// multiline\n// comment", commentBlock("some\nmultiline\ncomment\n"))
	require.Equal(t, "// single line comment", commentBlock("single line comment"))
}

func TestSetEndpoint(t *testing.T) {
	valid := endpoint{
		OperationID: "TestOperation",
		Group:       "TestGroup",
	}

	path := "/some/path"
	method := http.MethodGet
	cases := []struct {
		name   string
		e      *endpoint
		opts   Options
		exp    templateCtx
		expErr string
	}{
		{
			name: "returns populated dictionaries for a valid endpoint",
			e:    &valid,
			exp: templateCtx{
				Groups: map[string]*handlerGroup{
					valid.Group: &handlerGroup{Endpoints: []endpoint{valid}},
				},
				PathsByGroups: map[string]*pathsInGroup{
					valid.Group: &pathsInGroup{
						AllowedMethodsByPaths: map[string]*methodsInPath{
							path: &methodsInPath{
								OperationsByMethods: map[string]string{
									method: valid.OperationID,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "does not populate if the endpoint is nil",
			exp: templateCtx{
				Groups:        make(map[string]*handlerGroup),
				PathsByGroups: make(map[string]*pathsInGroup),
			},
		},
		{
			name: "does not populate if the endpoint has no group",
			e:    &endpoint{OperationID: "some"},
			exp: templateCtx{
				Groups:        make(map[string]*handlerGroup),
				PathsByGroups: make(map[string]*pathsInGroup),
			},
		},
		{
			name: "does not populate if the endpoint has no operation ID",
			e:    &endpoint{Group: "some"},
			exp: templateCtx{
				Groups:        make(map[string]*handlerGroup),
				PathsByGroups: make(map[string]*pathsInGroup),
			},
		},
		{
			name:   "returns error if opts.FailNoGroup = true and the endpoint has no group",
			e:      &endpoint{OperationID: "some"},
			opts:   Options{FailNoGroup: true},
			expErr: fmt.Sprintf("`%s %s` does not have the `x-handler-group` value", method, path),
		},
		{
			name:   "returns error if opts.FailNoOperationID = true and the endpoint has no operation ID",
			e:      &endpoint{Group: "some"},
			opts:   Options{FailNoOperationID: true},
			expErr: fmt.Sprintf("`%s %s` does not have the `operationId` value", method, path),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := &templateCtx{
				Groups:        make(map[string]*handlerGroup),
				PathsByGroups: make(map[string]*pathsInGroup),
			}

			err := setEndpoint(out, tc.opts, method, path, tc.e)
			if tc.expErr != "" {
				require.Error(t, err)
				require.Equal(t, tc.expErr, err.Error())
				return
			}

			require.EqualValues(t, tc.exp, *out)
		})
	}
}

func TestCreateTemplateCtx(t *testing.T) {
	spec := spec{
		Info: info{
			Title:       "Title",
			Description: "Description",
			Version:     "Version",
		},
		Paths: map[string]path{
			"/some/path": path{
				GET: &endpoint{
					Summary:     "GET Summary",
					Description: "GET Description",
					OperationID: "GETOperationID",
					Group:       "Group",
				},
				POST: &endpoint{
					Summary:     "POST Summary",
					Description: "POST Description",
					OperationID: "POSTOperationID",
					Group:       "Group",
				},
				PUT: &endpoint{
					Summary:     "PUT Summary",
					Description: "PUT Description",
					OperationID: "PUTOperationID",
					Group:       "Group",
				},
				PATCH: &endpoint{
					Summary:     "PATCH Summary",
					Description: "PATCH Description",
					OperationID: "PATCHOperationID",
					Group:       "Group",
				},
				DELETE: &endpoint{
					Summary:     "DELETE Summary",
					Description: "DELETE Description",
					OperationID: "DELETEOperationID",
					Group:       "Group",
				},
			},
			"/another/path": path{},
		},
	}

	out, err := createTemplateCtx(spec, Options{PackageName: "Test"})
	require.NoError(t, err)
	expected := templateCtx{
		PackageName: "Test",
		Spec:        spec,
		Groups: map[string]*handlerGroup{
			"Group": &handlerGroup{
				Endpoints: []endpoint{
					*spec.Paths["/some/path"].GET,
					*spec.Paths["/some/path"].POST,
					*spec.Paths["/some/path"].PUT,
					*spec.Paths["/some/path"].PATCH,
					*spec.Paths["/some/path"].DELETE,
				},
			},
		},
		PathsByGroups: map[string]*pathsInGroup{
			"Group": &pathsInGroup{
				AllowedMethodsByPaths: map[string]*methodsInPath{
					"/some/path": &methodsInPath{
						OperationsByMethods: map[string]string{
							"GET":    "GETOperationID",
							"POST":   "POSTOperationID",
							"PUT":    "PUTOperationID",
							"PATCH":  "PATCHOperationID",
							"DELETE": "DELETEOperationID",
						},
					},
				},
			},
		},
	}
	require.EqualValues(t, expected, out)
}
