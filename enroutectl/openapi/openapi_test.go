// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package openapi

import (
	"github.com/go-openapi/spec"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/saarasio/enroute/enroutectl/config"
	"testing"
)

type Assert struct {
	t *testing.T
}

func Equal(t *testing.T, want, got interface{}) {
	t.Helper()
	Assert{t}.Equal(want, got)
}

// Equal will call t.Fatal if want and got are not equal.
func (a Assert) Equal(want, got interface{}) {
	a.t.Helper()
	opts := []cmp.Option{
		cmpopts.AcyclicTransformer("UnmarshalAny", unmarshalAny),
		// errors to be equal only if both are nil or both are non-nil.
		cmp.Comparer(func(x, y error) bool {
			return (x == nil) == (y == nil)
		}),
	}
	diff := cmp.Diff(want, got, opts...)
	if diff != "" {
		a.t.Fatal(diff)
	}
}

func unmarshalAny(a *any.Any) proto.Message {
	if a == nil {
		return nil
	}
	pb, err := ptypes.Empty(a)
	if err != nil {
		panic(err.Error())
	}
	err = ptypes.UnmarshalAny(a, pb)
	if err != nil {
		panic(err.Error())
	}
	return pb
}

func TestConvertPathWithParamsToRegexPath(t *testing.T) {
	tests := map[string]struct {
		p_in        string
		query_param bool
		want        string
	}{
		"Path With One Param": {
			p_in:        `/pets/{petsId}`,
			query_param: false,
			want:        `/pets/(?P<petsId>.*)$`,
		},
		"Path With Two Params": {
			p_in:        `/pets/{petsId}/mushak/{mushakpet}`,
			query_param: false,
			want:        `/pets/(?P<petsId>.*)/mushak/(?P<mushakpet>.*)$`,
		},
		"Path With One Param w/ query param": {
			p_in:        `/pets/{petsId}`,
			query_param: true,
			want:        `/pets/(?P<petsId>.*)`,
		},
		"Path With Two Params w/ query param": {
			p_in:        `/pets/{petsId}/mushak/{mushakpet}`,
			query_param: true,
			want:        `/pets/(?P<petsId>.*)/mushak/(?P<mushakpet>.*)`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := pathWithParamsToRegexPath(tc.p_in, tc.query_param)
			Equal(t, tc.want, got)
		})
	}
}

func TestWalkSpec(t *testing.T) {

	var petstore200 = `
{
  "swagger": "2.0",
  "info": {
    "version": "1.0.0",
    "title": "Swagger Petstore",
    "license": {
      "name": "MIT"
    }
  },
  "host": "petstore.swagger.io",
  "basePath": "/v1",
  "schemes": [
    "http"
  ],
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/pets": {
      "get": {
        "summary": "List all pets",
        "operationId": "listPets",
        "tags": [
          "pets"
        ],
        "parameters": [
          {
            "name": "limit",
            "in": "query",
            "description": "How many items to return at one time (max 100)",
            "required": false,
            "type": "integer",
            "format": "int32"
          }
        ],
        "responses": {
          "200": {
            "description": "An paged array of pets",
            "headers": {
              "x-next": {
                "type": "string",
                "description": "A link to the next page of responses"
              }
            },
            "schema": {
              "$ref": "#/definitions/Pets"
            }
          },
          "default": {
            "description": "unexpected error",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          }
        }
      },
      "post": {
        "summary": "Create a pet",
        "operationId": "createPets",
        "tags": [
          "pets"
        ],
        "responses": {
          "201": {
            "description": "Null response"
          },
          "default": {
            "description": "unexpected error",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          }
        }
      }
    },
    "/pets/{petId}/abcd/{petId2}/efgh/{petId3}": {
      "get": {
        "summary": "Info for a specific pet",
        "operationId": "showPetById",
        "tags": [
          "pets"
        ],
        "parameters": [
          {
            "name": "petId",
            "in": "path",
            "required": true,
            "description": "The id of the pet to retrieve",
            "type": "string"
          }
        ],
        "responses": {
          "200": {
            "description": "Expected response to a valid request",
            "schema": {
              "$ref": "#/definitions/Pets"
            }
          },
          "default": {
            "description": "unexpected error",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Pet": {
      "required": [
        "id",
        "name"
      ],
      "properties": {
        "id": {
          "type": "integer",
          "format": "int64"
        },
        "name": {
          "type": "string"
        },
        "tag": {
          "type": "string"
        }
      }
    },
    "Pets": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/Pet"
      }
    },
    "Error": {
      "required": [
        "code",
        "message"
      ],
      "properties": {
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        }
      }
    }
  }
}
`

	var ps_json_want = `{"data":{"saaras_db_proxy":[{"proxy_name":"gw","proxy_services":[{"service":{"fqdn":"*","service_name":"openapi-petstore.swagger.io","routes":[{"route_config":"{\"prefix\":\"/v1/pets\",\"header\":[{\"name\":\":method\",\"exact\":\"POST\"}]}\n","route_name":"openapi-createPets-v1pets-c7baa9ec7f","route_upstreams":[{"upstream":{"upstream_ip":"petstore.swagger.io","upstream_name":"openapi-upstream-petstore.swagger.io","upstream_port":80}}]},{"route_config":"{\"prefix\":\"/v1/pets\",\"header\":[{\"name\":\":method\",\"exact\":\"GET\"}]}\n","route_name":"openapi-listPets-v1pets-ec51ad447d","route_upstreams":[{"upstream":{"upstream_ip":"petstore.swagger.io","upstream_name":"openapi-upstream-petstore.swagger.io","upstream_port":80}}]},{"route_config":"{\"prefix\":\"/v1/pets/(?P<petId>.*)/abcd/(?P<petId2>.*)/efgh/(?P<petId3>.*)$\",\"header\":[{\"name\":\":method\",\"exact\":\"GET\"}]}\n","route_name":"openapi-showPetById-f06a8d1fd1-dc96a9cc81","route_upstreams":[{"upstream":{"upstream_ip":"petstore.swagger.io","upstream_name":"openapi-upstream-petstore.swagger.io","upstream_port":80}}]}]}}]}]}}
`
	tests := map[string]struct {
		spec_in           string
		want_enroute_json string
	}{
		"Convert Spec to Enroute Cfg": {
			spec_in:           petstore200,
			want_enroute_json: ps_json_want,
		},
	}

	for name, tc := range tests {
		var ecfg config.EnrouteConfig
		t.Run(name, func(t *testing.T) {
			l, _ := JSONSpec(tc.spec_in, true)
			SwaggerToEnroute(l.Spec(), &ecfg)
			got_enroute_json, _ := JSONMarshal(ecfg)
			Equal(t, tc.want_enroute_json, string(got_enroute_json))
		})
	}
}

func TestSwaggerPathSpecToSaarasRoute(t *testing.T) {
	tests := map[string]struct {
		s_in      spec.Swagger
		rname_in  string
		opname_in string
		prefix_in string
		want      config.Routes
	}{
		"Simple": {
			s_in: spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Host: "testhost",
				},
			},
			rname_in:  "test",
			opname_in: "GET",
			prefix_in: "/test",
			want: config.Routes{
				RouteConfig: "{\"prefix\":\"/test\",\"header\":[{\"name\":\":method\",\"exact\":\"GET\"}]}\n",
				RouteName:   "openapi-test-test-89c114d0f6",
				RouteUpstreams: []config.RouteUpstreams{{
					Upstream: config.Upstream{
						UpstreamName: "openapi-upstream-testhost",
						UpstreamIP:   "testhost",
						UpstreamPort: 80,
					},
				}},
			},
		},
		"Missing Route Name": {
			s_in: spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Host: "testhost",
				},
			},
			rname_in:  "",
			opname_in: "GET",
			prefix_in: "/test2",
			want: config.Routes{
				RouteConfig: "{\"prefix\":\"/test2\",\"header\":[{\"name\":\":method\",\"exact\":\"GET\"}]}\n",
				RouteName:   "openapi-test2-test2-7a669d3de6",
				RouteUpstreams: []config.RouteUpstreams{{
					Upstream: config.Upstream{
						UpstreamName: "openapi-upstream-testhost",
						UpstreamIP:   "testhost",
						UpstreamPort: 80,
					},
				}},
			},
		},
		"Method PUT": {
			s_in: spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Host: "testhost",
				},
			},
			rname_in:  "put",
			opname_in: "PUT",
			prefix_in: "/test",
			want: config.Routes{
				RouteConfig: "{\"prefix\":\"/test\",\"header\":[{\"name\":\":method\",\"exact\":\"PUT\"}]}\n",
				RouteName:   "openapi-put-test-818f8f196d",
				RouteUpstreams: []config.RouteUpstreams{{
					Upstream: config.Upstream{
						UpstreamName: "openapi-upstream-testhost",
						UpstreamIP:   "testhost",
						UpstreamPort: 80,
					},
				}},
			},
		},
	}

	s := spec.Swagger{}
	s.SwaggerProps.Host = "testhost"

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := swaggerPathSpecToSaarasRoute(&tc.s_in, tc.rname_in, tc.opname_in, tc.prefix_in)
			Equal(t, tc.want, got)
		})
	}
}
