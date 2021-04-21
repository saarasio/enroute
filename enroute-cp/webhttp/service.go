// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package webhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/saarasio/enroute/enroute-dp/saaras"
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Route struct {
	Route_name   string `json:"route_name" xml:"route_name" form:"route_name" query:"route_name"`
	Route_prefix string `json:"route_prefix" xml:"route_prefix" form:"route_prefix" query:"route_prefix"`

	// Route_config holds configuration in json. Use this config if present. Else, fallback to individual fields above
	Route_config string `json:"route_config" xml:"route_config" form:"route_config" query:"route_config"`
}

type Service struct {
	Service_name string `json:"service_name" xml:"service_name" form:"service_name" query:"service_name"`
	Fqdn         string `json:"fqdn" xml:"fqdn" form:"fqdn" query:"fqdn"`

	// Service_config holds configuration in json. Use this config if present. Else, fallback to individual fields above
	Service_config string `json:"service_config" xml:"service_config" form:"service_config" query:"service_config"`
}

var QDeleteService = `
mutation delete_service($service_name: String!) {
  delete_saaras_db_service(where: {service_name: {_eq: $service_name}}) {
    affected_rows
  }
}
`

var QServiceProxy = `
query get_proxy_service($service_name: String!) {
  saaras_db_proxy(where: {proxy_services: {service: {service_name: {_eq: $service_name}}}}) {
    proxy_id
    proxy_name
    update_ts
    create_ts
  }
}

`

var QServiceRoutes = `
query get_service_routes($service_name: String!){
  saaras_db_route(where: {service: {service_name: {_eq: $service_name}}}) {
    route_id
    route_name
    route_prefix
    config_json
    create_ts
    update_ts
    route_filters {
      filter {
        filter_name
        filter_type
      }
    }
  }
}
`

var QDeleteRouteOneRoute = `
mutation delete_service_route($route_name: String!, $service_name: String!) {
  delete_saaras_db_route(
    where: {
      _and: {
        service: {service_name: {_eq: $service_name}}, 
        route_name: {_eq: $route_name}}
    }) {
    affected_rows
  }
}
`

var QGetServiceRouteUpstreams = `
query get_upstream($service_name: String!, $route_name: String!) {
  saaras_db_route(where: {_and: {service: {service_name: {_eq: $service_name}}, route_name: {_eq: $route_name}}}) {
    route_name
    route_prefix
    route_upstreams {
      upstream {
        upstream_name
        upstream_ip
        upstream_port
        create_ts
        update_ts
      }
    }
  }
}
`

var QGetServiceSecret = `
query get_upstream($service_name: String!) {
  saaras_db_secret(where: {service_secrets: {service: {service_name: {_eq: $service_name}}}}) {
    secret_id
    secret_name
    artifacts {
      artifact_id
      artifact_name
      artifact_type
      artifact_value
    }
  }
}
`

var QAssociateServiceSecret = `
mutation insert_service_secret($service_name: String!, $secret_name: String!) {
  insert_saaras_db_service_secret(
    objects: {
      service: {data: {service_name: $service_name}, 
        on_conflict: {constraint: service_service_name_key, update_columns: update_ts}}, 
      secret: {data: {secret_name: $secret_name}, 
        on_conflict: {constraint: secret_secret_name_key, update_columns: update_ts}}
    }, on_conflict: {constraint: service_secret_service_id_secret_id_key, update_columns: update_ts}) {
    affected_rows
  }
}
`

var QDisassociateServiceSecret = `
mutation delete_service_secret($service_name: String!, $secret_name: String!) {
  delete_saaras_db_service_secret(
    where: 
    {
      _and: 
      {
        secret: {secret_name: {_eq: $secret_name}}, 
        service: {service_name: {_eq: $service_name}}
      }
    }
  ) {
    affected_rows
  }
}
`

var QAssociateRouteUpstream = `
mutation insert_route_upstream(
  $service_name: String!, 
  $route_name: String!, 
  $upstream_name: String!) 
  {
      insert_saaras_db_route_upstream(
      objects: 
      {
        upstream: {data: {upstream_name: $upstream_name}, 
          on_conflict: {constraint: upstream_upstream_name_key, update_columns: update_ts}}, 
        route: 
        {
          data: 
          {
            route_name: $route_name, 
            service: 
            {
              data: 
              {
                service_name: $service_name
              }, 
              on_conflict: {constraint: service_service_name_key, update_columns: update_ts}
            }
          }, 
          on_conflict: {constraint: route_service_id_route_name_key, update_columns: update_ts}
        }
      }, 
      on_conflict: {constraint: route_upstream_route_id_upstream_id_key, update_columns: update_ts}) {
    affected_rows
  }
}
`

var QDeleteServiceRouteUpstreamAssociation = `
mutation delete_route_upstream($service_name: String!, $route_name: String!, $upstream_name: String!) {
  delete_saaras_db_route_upstream(where: {
    _and: {
      route: 
      {
        _and: {
          route_name: {_eq: $route_name}, 
          service: {service_name: {_eq: $service_name}}}
      }, 
      upstream: 
      {
        upstream_name: {_eq: $upstream_name}
      }
    }
  }
  ) {
    affected_rows
  }
}
`

var QGetOneServiceDetail = `
query get_upstream($service_name: String!) {
  saaras_db_service(where: {service_name: {_eq: $service_name}}) {
    service_id
    service_name
    fqdn
    create_ts
    routes {
      route_id
      route_name
      route_upstreams {
        upstream {
          upstream_id
          upstream_name
          upstream_ip
          upstream_port
        }
      }
      route_filters {
        filter {
          filter_name
          filter_type
        }
      }
      route_prefix
    }
    service_secrets {
      secret {
        secret_id
        secret_name
        secret_sni
        secret_key
        secret_cert
      }
    }
    service_filters {
      filter {
        filter_name
        filter_type
      }
    }
  }
}
`

func GetURL() string {
	return "http://" + HOST + ":" + PORT + "/v1/graphql"
}

func db_update_service(s *Service, log *logrus.Entry) (int, string) {

	var QPatchService = `
	mutation update_service(
		$service_name : String!,
		$fqdn : String!
	){
	  update_saaras_db_service
		(
			
			where: {service_name: {_eq: $service_name}}, 
	
			_set: 
	
			{
				fqdn: $fqdn
			}
	
		) {
	    affected_rows
	  }
	}
`

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = s.Service_name
	args["fqdn"] = s.Fqdn

	if err := saaras.RunDBQuery(url, QPatchService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	return http.StatusCreated, buf.String()

}

// @Summary Update a service
// @Description Update a service
// @Tags service
// @Accept  json
// @Produce  json
// @Param Service body webhttp.Service true "Fields of service to update"
// @Param service_name path string true "name of service to update"
// @Success 201 {number} uint OK
// @Router /service/{service_name} [patch]
// @Security ApiKeyAuth
func PATCH_Service(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Service)
	if err := c.Bind(s); err != nil {
		return err
	}

	log.Infof("Received service [%v]\n", s)

	service_name := c.Param("service_name")

	code, buf, s_in_db := db_get_one_service(service_name, true, log)
	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	// For service, right now, only Fqdn can be patched.
	// Ensure that we are passed a valid Fqdn
	if len(s.Fqdn) == 0 {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Please provide fqdn using Fqdn field\"}")
	}

	// Overwrite Fqdn value
	s_in_db.Fqdn = s.Fqdn

	code2, buf2 := db_update_service(s_in_db, log)

	return c.JSONBlob(code2, []byte(buf2))
}

func db_insert_service(s *Service, log *logrus.Entry) (int, string) {

	var QCreateService = `
mutation insert_service($fqdn: String!, $service_name: String!) {
  insert_saaras_db_service(
    objects: 
    {
      service_name: $service_name, 
      fqdn: $fqdn
    }, on_conflict: {
      constraint: service_service_name_key, update_columns: update_ts
    }
  ) {
    affected_rows
  }
}
`

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = s.Service_name
	args["fqdn"] = s.Fqdn

	if err := saaras.RunDBQuery(url, QCreateService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	return http.StatusCreated, buf.String()
}

func db_get_service(log *logrus.Entry) (int, string) {
	var QGetService = `
query get_service {
  saaras_db_service {
    service_id
    service_name
    fqdn
    create_ts
    update_ts
  }
}
`
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QGetService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	return http.StatusOK, buf.String()
}

func db_get_one_service(service_name string, decode bool, log *logrus.Entry) (int, string, *Service) {

	var QGetOneService = `
query get_one_service($service_name : String!) {
  saaras_db_service (where: {service_name: {_eq: $service_name}}) {
    service_id
    service_name
    fqdn
    create_ts
    update_ts
  }
}
`
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	args["service_name"] = service_name
	if err := saaras.RunDBQuery(url, QGetOneService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String(), nil
	}

	//{
	//    "data": {
	//        "saaras_db_service": [
	//            {
	//                "create_ts": "2019-09-05T01:57:45.174459+00:00",
	//                "fqdn": "testfqdn3.com",
	//                "service_id": 1,
	//                "service_name": "test",
	//                "update_ts": "2019-09-05T04:11:26.619627+00:00"
	//            }
	//        ]
	//    }
	//}

	// Note: To auto-generate a golang data structure paste the above here:
	// https://mholt.github.io/json-to-go/

	type OneSaarasDBService struct {
		Service_id   int64  `json:"service_id"`
		Service_name string `json:"service_name"`
		Fqdn         string `json:"fqdn"`
		Create_ts    string `json:"create_ts"`
		Update_ts    string `json:"update_ts"`
	}

	type SaarasDBService struct {
		Saaras_db_service []OneSaarasDBService `json:"saaras_db_service"`
	}

	type ServiceResponse struct {
		Data SaarasDBService `json:"data"`
	}

	var s Service

	if decode {
		var gr ServiceResponse
		log.Debugf("Decoding :\n %s\n", buf.String())
		if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
			errors.Wrap(err, "decoding response")
			log.Errorf("Error when decoding json [%v]\n", err)
			return http.StatusBadRequest, buf.String(), nil
		}

		if len(gr.Data.Saaras_db_service) > 0 {
			s.Fqdn = gr.Data.Saaras_db_service[0].Fqdn
			s.Service_name = gr.Data.Saaras_db_service[0].Service_name
		}
	}

	return http.StatusOK, buf.String(), &s
}

func db_copy_service(service_name_src string, service_name_dst string, log *logrus.Entry) (int, string) {

	code, buf, s := db_get_one_service(service_name_src, true, log)

	if code != http.StatusOK {
		return code, buf
	}
	// Overwrite service name to that of destination
	s.Service_name = service_name_dst
	code2, buf2 := db_insert_service(s, log)
	return code2, buf2
}

func validate_service(s *Service) (int, string) {
	if len(s.Service_name) == 0 {
		return http.StatusBadRequest, "{\"Error\" : \"Please provide service name using Service_name field\"}"
	}

	if len(s.Fqdn) == 0 {
		return http.StatusBadRequest, "{\"Error\" : \"Please provide fqdn using Fqdn field\"}"
	}

	return http.StatusOK, ""
}

// @Summary Copy service
// @Tags service, operational-verbs
// @Accept  json
// @Produce  json
// @Param service_name_src path string true "Name of service"
// @Param service_name_dst path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/copy/{service_name_src}/{service_name_dst} [post]
// @Security ApiKeyAuth
func POST_Service_Copy(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name_src := c.Param("service_name_src")
	service_name_dst := c.Param("service_name_dst")

	code, buf := db_copy_service(service_name_src, service_name_dst, log)

	return c.JSONBlob(code, []byte(buf))
}

// @Summary Deep copy service
// @Description Deep copy service copies service, associated routes and upstream associations. Upstream copies are not created.
// @Tags service, route, operational-verbs
// @Accept  json
// @Produce  json
// @Param service_name_src path string true "Name of service"
// @Param service_name_dst path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/deepcopy/{service_name_src}/{service_name_dst} [post]
// @Security ApiKeyAuth
func POST_Service_DeepCopy(c echo.Context) error {
	// Copy service
	// Copy routes for service
	// Copy upstream for routes
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name_src := c.Param("service_name_src")
	service_name_dst := c.Param("service_name_dst")

	code, buf, p := db_get_service_details(service_name_src, log)
	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	log.Infof("POST_Service_DeepCopy() Got service detail [%+v]\n", p)

	if len(p.Data.SaarasDbService) > 0 {
		// Copy service using
		log.Infof("POST_Service_DeepCopy() Copy service [%s] to [%s]\n", service_name_src, service_name_dst)
		code, buf := db_copy_service(service_name_src, service_name_dst, log)
		if code != http.StatusCreated {
			return c.JSONBlob(code, []byte(buf))
		}

		log.Infof("POST_Service_DeepCopy() Copy service result [%s]\n", buf)

		for _, oneRoute := range p.Data.SaarasDbService[0].Routes {

			// Copy service
			code, buf, r_in_db := db_get_one_service_route(service_name_src, oneRoute.RouteName, true, log)
			if code != http.StatusOK {
				return c.JSONBlob(code, []byte(buf))
			}

			log.Infof("POST_Service_DeepCopy() Got route detail [%+v]\n", r_in_db)

			// Copy routes for service
			code, buf = db_insert_service_route(service_name_dst, r_in_db, log)
			if code != http.StatusCreated {
				return c.JSONBlob(code, []byte(buf))
			}

			log.Infof("POST_Service_DeepCopy() Copy route result [%s]\n", buf)

			for _, oneRouteUpstream := range oneRoute.RouteUpstreams {
				u_name := oneRouteUpstream.Upstream.UpstreamName
				code, buf := db_associate_service_route_upstream(service_name_dst, oneRoute.RouteName, u_name, log)
				log.Infof("POST_Service_DeepCopy() Associate upstream result [%s]\n", buf)
				if code != http.StatusCreated {
					return c.JSONBlob(code, []byte(buf))
				}
			}
		}
	}

	return c.JSONBlob(http.StatusCreated, []byte("{\"affected_rows\":1}"))
}

// @Summary Create a service
// @Description Create a service
// @Tags service
// @Accept  json
// @Produce  json
// @Param Service body webhttp.Service true "Service to create"
// @Success 201 {number} uint OK
// @Router /service [post]
// @Security ApiKeyAuth
func POST_Service(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Service)
	if err := c.Bind(s); err != nil {
		return err
	}

	code_validate, buf_validate := validate_service(s)

	if code_validate != http.StatusOK {
		return c.JSONBlob(code_validate, []byte(buf_validate))
	}

	code, buf := db_insert_service(s, log)
	return c.JSONBlob(code, []byte(buf))
}

// @Summary List all services
// @Description Get a list of all services for all proxies
// @Tags service
// @Accept  json
// @Produce  json
// @Success 200 {number} uint OK
// @Router /service [get]
// @Security ApiKeyAuth
func GET_Service(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	code, buf := db_get_service(log)
	return c.JSONBlob(code, []byte(buf))
}

// @Summary Fetch some details of specified service
// @Description Fetch some details of specified service
// @Tags service
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/{service_name} [get]
// @Security ApiKeyAuth
func GET_One_Service(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	code, buf, _ := db_get_one_service(service_name, false, log)
	return c.JSONBlob(code, []byte(buf))
}

// @Summary Get status
// @Description Get status of HTTP service
// @Tags service
// @Accept  json
// @Produce  json
// @Success 200 {number} uint OK
// @Router /status/{status} [get]
// @Security ApiKeyAuth
func GET_Status(c echo.Context) error {

	ID = c.Param("status")

	rstr := "{\"Status\" : \"OK\"}"
	return c.JSONBlob(http.StatusOK, []byte(rstr))
}

// @Summary Fetch detail of specified service
// @Description Fetch detail of specified service
// @Tags service, operational-verbs
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/dump/{service_name} [get]
// @Security ApiKeyAuth
func GET_One_Service_Detail(c echo.Context) error {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	args["service_name"] = service_name
	if err := saaras.RunDBQuery(url, QGetOneServiceDetail, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Delete the specified service
// @Description Delete specified service
// @Tags service
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/{service_name} [delete]
// @Security ApiKeyAuth
func DELETE_Service(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	service_name := c.Param("service_name")
	args["service_name"] = service_name
	if err := saaras.RunDBQuery(url, QDeleteService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Fetch list of proxies on which this service is programmed
// @Description Fetch list of proxies on which this service is programmed
// @Tags service, proxy
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/proxy [get]
// @Security ApiKeyAuth
func GET_Service_Proxy(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	service_name := c.Param("service_name")
	args["service_name"] = service_name
	if err := saaras.RunDBQuery(url, QServiceProxy, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func db_update_service_route(service_name string, r *Route, log *logrus.Entry) (int, string) {

	var QPostUpdateServiceRoute = `
mutation update_route($service_name:String!, $route_name:String!, $route_prefix:String!){
  update_saaras_db_route
  (
    where: 
    {
      _and: 
      {
        service: {service_name: {_eq: $service_name}}, 
        route_name: {_eq: $route_name}
      }
    }, 
    _set: {route_prefix: $route_prefix}
  ) {
    affected_rows
  }
}
`
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["route_name"] = r.Route_name
	args["route_prefix"] = r.Route_prefix

	if err := saaras.RunDBQuery(url, QPostUpdateServiceRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	return http.StatusCreated, buf.String()
}

func setRouteConfigJson(log *logrus.Entry, route_config string, args *map[string]interface{}) {
	cfg, err := saarasconfig.UnmarshalRouteMatchCondition(route_config)
	if err == nil {
		(*args)["config_json"] = cfg
	} else {
		default_routematchconditions := saarasconfig.RouteMatchConditions{
			Prefix:          "/",
			MatchConditions: []saarasconfig.RouteMatchCondition{{}},
		}
		(*args)["config_json"] = default_routematchconditions
	}
}

func db_insert_service_route(service_name string, r *Route, log *logrus.Entry) (int, string) {

	var QPostServiceRoute = `
mutation insert_service_route($route_name: String!, $route_prefix: String, $service_name: String!, $route_config: String, $config_json: jsonb) {
  insert_saaras_db_route(
    objects: 
    {
      route_name: $route_name, 
      route_prefix: $route_prefix, 
      route_config: $route_config, 
      config_json: $config_json, 
      service: 
      {
        data: 
        {
          service_name: $service_name
        }, 
        on_conflict: {constraint: service_service_name_key, update_columns: service_name}
      }
    }
  ) 
  {
    affected_rows
  }
}
`
	var buf bytes.Buffer
	var args map[string]interface{}
	args = make(map[string]interface{})
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["route_config"] = r.Route_config
	args["route_name"] = r.Route_name
	args["route_prefix"] = r.Route_prefix
	args["service_name"] = service_name

	setRouteConfigJson(log, r.Route_config, &args)

	if err := saaras.RunDBQueryGenericVals(url, QPostServiceRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	return http.StatusCreated, buf.String()
}

func validate_service_route(r *Route) (int, string) {
	if len(r.Route_name) == 0 {
		return http.StatusBadRequest, "{\"Error\" : \"Please provide route name using Name field\"}"
	}

	// Look for config in Route_config (which is specified as json)
	// If absent, fallback to reading prefix from Route_prefix
	if len(r.Route_config) > 0 {
		routeConfig, err := saarasconfig.UnmarshalRouteMatchCondition(r.Route_config)
		if err != nil {
			return http.StatusBadRequest, fmt.Sprintf("{\"Error\" : \"Failed to unmarshal route config %s\"}", r.Route_config)
		}

		if routeConfig.Prefix != "" {
			if routeConfig.Prefix[0] != '/' {
				return http.StatusBadRequest, fmt.Sprintf("{\"Error\" : \"Prefix condition must start with /\"}")
			}
		}

	} else {
		if len(r.Route_prefix) == 0 {
			return http.StatusBadRequest, "{\"Error\" : \"Please provide route prefix using Prefix field\"}"
		}
	}

	return http.StatusOK, "{}"
}

// @Summary Create a route associated with a service
// @Description Create a route associated with a service
// @Tags service, route
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param Route body webhttp.Route true "Route to create"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route [post]
// @Security ApiKeyAuth
func POST_Service_Route(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	r := new(Route)
	if err := c.Bind(r); err != nil {
		return err
	}

	service_name := c.Param("service_name")

	code, buf := validate_service_route(r)
	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	code2, buf2 := db_insert_service_route(service_name, r, log)

	return c.JSONBlob(code2, []byte(buf2))
}

// @Summary Update a route associated with a service
// @Description Update a route associated with a service
// @Tags service, route
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of service"
// @Param Route body webhttp.Route true "Route to update"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route/{route_name} [patch]
// @Security ApiKeyAuth
func PATCH_Service_Route(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	r := new(Route)
	if err := c.Bind(r); err != nil {
		return err
	}

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")

	log.Infof("Fetch service from DB service [%s] route [%s] \n", service_name, route_name)
	code, buf, r_in_db := db_get_one_service_route(service_name, route_name, true, log)
	log.Infof("Fetched service from DB service [%+v]  \n", r_in_db)
	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	// Since only prefix is something that you can update in the route, overwrite the prefix
	if len(r.Route_prefix) == 0 {
		return c.JSONBlob(http.StatusBadRequest, []byte("{\"Error\" : \"Please provide route prefix using Route_Prefix field\"}"))
	}

	r_in_db.Route_prefix = r.Route_prefix

	code2, buf2 := validate_service_route(r_in_db)
	if code2 != http.StatusOK {
		return c.JSONBlob(code2, []byte(buf2))
	}

	log.Infof("Update route to [%+v]  \n", r_in_db)
	code3, buf3 := db_update_service_route(service_name, r_in_db, log)
	return c.JSONBlob(code3, []byte(buf3))
}

type ServiceDetail struct {
	Data struct {
		SaarasDbService []struct {
			ServiceName string `json:"service_name"`
			Routes      []struct {
				RouteName      string `json:"route_name"`
				RouteUpstreams []struct {
					Upstream struct {
						UpstreamName string `json:"upstream_name"`
					} `json:"upstream"`
				} `json:"route_upstreams"`
			} `json:"routes"`
			ServiceSecrets []interface{} `json:"service_secrets"`
		} `json:"saaras_db_service"`
	} `json:"data"`
}

func db_get_service_details(service_name string, log *logrus.Entry) (int, string, *ServiceDetail) {

	var QGetServiceDetail = `
query get_saaras_db_proxy_names($service_name: String!) {
  saaras_db_service(where: {service_name: {_eq: $service_name}}) {
    service_name
    routes {
      route_name
      route_upstreams {
        upstream {
          upstream_name
        }
      }
    }
    service_secrets {
      secret {
        secret_name
      }
    }
  }
}
`

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	args["service_name"] = service_name
	if err := saaras.RunDBQuery(url, QGetServiceDetail, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String(), nil
	}

	log.Infof("db_get_service_details(): Q Response: [%+v]\n", buf.String())

	var gr ServiceDetail
	log.Debugf("Decoding :\n %s\n", buf.String())
	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		errors.Wrap(err, "decoding response")
		log.Errorf("Error when decoding json [%v]\n", err)
	}

	log.Infof("db_get_service_details(): Decoded Response: [%+v]\n", gr)

	return http.StatusOK, "", &gr
}

// @Summary Copy a route on source service to a destination service
// @Description Copy a route on source service to a destination service
// @Tags service, route, operational-verbs
// @Accept  json
// @Produce  json
// @Param service_name_src path string true "Name of service"
// @Param service_name_dst path string true "Name of service"
// @Param route_name path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/copyroute/{service_name_src}/{service_name_dst}/route/{route_name} [post]
// @Security ApiKeyAuth
func POST_Service_Route_Copy(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name_src := c.Param("service_name_src")
	service_name_dst := c.Param("service_name_dst")
	route_name := c.Param("route_name")

	code, buf, r := db_get_one_service_route(service_name_src, route_name, true, log)
	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	code2, buf2 := validate_service_route(r)
	if code2 != http.StatusOK {
		return c.JSONBlob(code, []byte(buf2))
	}

	log.Debugf("Inserting route [%+v] in service [%s]\n", r, service_name_dst)
	code3, buf3 := db_insert_service_route(service_name_dst, r, log)
	return c.JSONBlob(code3, []byte(buf3))
}

// @Summary Get all routes for a service
// @Description Get all routes for a service
// @Tags service, route
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route [get]
// @Security ApiKeyAuth
func GET_Service_Route(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	args["service_name"] = service_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QServiceRoutes, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	buf2 := bytes.Replace(buf.Bytes(), []byte("config_json"), []byte("route_config"), -1)

	return c.JSONBlob(http.StatusOK, buf2)
}

func db_get_one_service_route(service_name string, route_name string, decode bool, log *logrus.Entry) (int, string, *Route) {

	var QServiceRouteOneRoute = `
 	query get_service_routes($service_name: String!, $route_name: String!) {
        saaras_db_route(
            where: {
                _and: {
                    route_name: {_eq: $route_name},
                    service: {service_name: {_eq: $service_name}}
                }
            }) {
                route_id
                route_name
                route_prefix
                create_ts
                update_ts
                        route_filters {
                  filter {
                    filter_name
                    filter_type
                  }
                }
            }
        }
	`

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	args["service_name"] = service_name
	args["route_name"] = route_name
	if err := saaras.RunDBQuery(url, QServiceRouteOneRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String(), nil
	}

	// Note: To auto-generate a golang data structure use:
	// https://mholt.github.io/json-to-go/
	type SaarasDbRoute struct {
		RouteID     int       `json:"route_id"`
		RouteName   string    `json:"route_name"`
		RoutePrefix string    `json:"route_prefix"`
		CreateTs    time.Time `json:"create_ts"`
		UpdateTs    time.Time `json:"update_ts"`
	}
	type Data struct {
		SaarasDbRoute []SaarasDbRoute `json:"saaras_db_route"`
	}
	type AutoGenerated struct {
		Data Data `json:"data"`
	}

	var r Route

	if decode {
		var gr AutoGenerated
		log.Debugf("Decoding :\n %s\n", buf.String())
		if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
			errors.Wrap(err, "decoding response")
			log.Errorf("Error when decoding json [%v]\n", err)
		}

		log.Debugf("Decoded json payload [%+v]\n", gr)
		if len(gr.Data.SaarasDbRoute) > 0 {
			r.Route_name = gr.Data.SaarasDbRoute[0].RouteName
			r.Route_prefix = gr.Data.SaarasDbRoute[0].RoutePrefix
		}
	}

	log.Debugf("Returing decoded route [%+v]\n", r)
	return http.StatusOK, buf.String(), &r

}

// @Summary Get information for a route for given service
// @Description Get information for a route for given service
// @Tags service, route
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of route"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route/{route_name} [get]
// @Security ApiKeyAuth
func GET_Service_Route_OneRoute(c echo.Context) error {
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")

	args["service_name"] = service_name
	args["route_name"] = route_name

	code, buf, _ := db_get_one_service_route(service_name, route_name, false, log)

	return c.JSONBlob(code, []byte(buf))
}

// @Summary Delete the route associated with the service
// @Description Delete the route associated with the service
// @Tags service, route
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of route to delete"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route/{route_name} [get]
// @Security ApiKeyAuth
func DELETE_Service_Route_OneRoute(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")

	args["service_name"] = service_name
	args["route_name"] = route_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QDeleteRouteOneRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func db_associate_service_route_upstream(service_name, route_name, upstream_name string, log *logrus.Entry) (int, string) {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	args["service_name"] = service_name
	args["route_name"] = route_name
	args["upstream_name"] = upstream_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QAssociateRouteUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return http.StatusCreated, buf.String()
}

// @Summary Associate an upstream with a service route
// @Description Associate an upstream with a service route
// @Tags service, route, upstream
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of route"
// @Param upstream_name path string true "Name of upstream to associate with service route"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route/{route_name}/upstream/{upstream_name} [get]
// @Security ApiKeyAuth
func POST_Service_Route_Upstream_Associate(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")
	upstream_name := c.Param("upstream_name")

	code, buf := db_associate_service_route_upstream(service_name, route_name, upstream_name, log)
	return c.JSONBlob(code, []byte(buf))
}

// @Summary Get all upstreams associated with a service route
// @Description Get all upstreams associated with a service route
// @Tags service, route, upstream
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of route"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route/{route_name}/upstream [get]
// @Security ApiKeyAuth
func GET_Service_Route_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")

	args["service_name"] = service_name
	args["route_name"] = route_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QGetServiceRouteUpstreams, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Disassociate an upstream with a service route
// @Description Disassociate an upstream with a service route. This does not delete the upstream.
// @Tags service, route, upstream
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of route"
// @Param upstream_name path string true "Name of upstream to disassociate with service route"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/route/{route_name}/upstream/{upstream_name} [get]
// @Security ApiKeyAuth
func DELETE_Service_Route_Upstream_Associate(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")
	upstream_name := c.Param("upstream_name")

	args["service_name"] = service_name
	args["route_name"] = route_name
	args["upstream_name"] = upstream_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QDeleteServiceRouteUpstreamAssociation, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary List all secrets associated with a service
// @Description List all secrets associated with a service
// @Tags service, secret
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/secret [get]
// @Security ApiKeyAuth
func GET_Service_Secret(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")

	args["service_name"] = service_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QGetServiceSecret, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Associate a secret with a service
// @Description Associate a secret with a service
// @Tags service, secret
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param secret_name path string true "Name of secret"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/secret/{secret_name} [post]
// @Security ApiKeyAuth
func POST_Service_Secret(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	secret_name := c.Param("secret_name")

	args["service_name"] = service_name
	args["secret_name"] = secret_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QAssociateServiceSecret, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Disassociate a secret with a service
// @Description Disassociate a secret with a service
// @Tags service, secret
// @Accept  json
// @Produce  json
// @Param service_name path string true "Name of service"
// @Param secret_name path string true "Name of secret"
// @Success 200 {number} uint OK
// @Router /service/{service_name}/secret/{secret_name} [delete]
// @Security ApiKeyAuth
func DELETE_Service_Secret(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	secret_name := c.Param("secret_name")

	args["service_name"] = service_name
	args["secret_name"] = secret_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QDisassociateServiceSecret, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func Add_service_routes(e *echo.Echo) {

	// Service CRUD
	e.GET("/service", GET_Service)
	e.GET("/service/:service_name", GET_One_Service)
	e.POST("/service", POST_Service)
	e.PATCH("/service/:service_name", PATCH_Service)
	e.DELETE("/service/:service_name", DELETE_Service)

	// Service to Proxy association
	e.GET("/service/:service_name/proxy", GET_Service_Proxy)

	// Route CRUD
	// Route is always associated to a service and isn't an independent entity.
	// Hence all Route operations are associated with a service
	e.POST("/service/:service_name/route", POST_Service_Route)
	e.PATCH("/service/:service_name/route/:route_name", PATCH_Service_Route)
	e.GET("/service/:service_name/route", GET_Service_Route)
	e.GET("/service/:service_name/route/:route_name", GET_Service_Route_OneRoute)
	e.DELETE("/service/:service_name/route/:route_name", DELETE_Service_Route_OneRoute)

	// Route to Upstream associations
	// Since Route is dependent on Service, this always ends up being Service_Route to Upstream association
	e.GET("/service/:service_name/route/:route_name/upstream", GET_Service_Route_Upstream)
	e.POST("/service/:service_name/route/:route_name/upstream/:upstream_name", POST_Service_Route_Upstream_Associate)
	e.DELETE("/service/:service_name/route/:route_name/upstream/:upstream_name", DELETE_Service_Route_Upstream_Associate)

	// Service to Secret associations
	e.GET("/service/:service_name/secret", GET_Service_Secret)
	e.POST("/service/:service_name/secret/:secret_name", POST_Service_Secret)
	e.DELETE("/service/:service_name/secret/:secret_name", DELETE_Service_Secret)

	// Service verb
	e.POST("/service/copy/:service_name_src/:service_name_dst", POST_Service_Copy)
	e.POST("/service/deepcopy/:service_name_src/:service_name_dst", POST_Service_DeepCopy)
	e.POST("/service/copyroute/:service_name_src/:service_name_dst/route/:route_name", POST_Service_Route_Copy)
	e.GET("/service/dump/:service_name", GET_One_Service_Detail)

	e.GET("/status/:status", GET_Status)
}
