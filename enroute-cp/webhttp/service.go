package webhttp

import (
	"bytes"
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/saarasio/enroute/saaras"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

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
    create_ts
    update_ts
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
  saaras_db_service(where: {_and: {service_name: {_eq: $service_name}, routes: {route_name: {_eq: $route_name}}}}) {
    routes {
      route_id
      route_name
      route_upstreams {
        upstream {
	  upstream_id
	  upstream_name
	  upstream_ip
	  upstream_port
	  upstream_hc_path
	  upstream_hc_host
	  upstream_hc_intervalseconds
	  upstream_hc_timeoutseconds
	  upstream_hc_unhealthythresholdcount
	  upstream_hc_healthythresholdcount
	  upstream_strategy
	  upstream_validation_cacertificate
	  upstream_validation_subjectname
	  create_ts
	  update_ts
        }
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

func PATCH_Service(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Service)
	if err := c.Bind(s); err != nil {
		return err
	}

	service_name := c.Param("service_name")

	code, buf, s_in_db := db_get_one_service(service_name, true, log)
	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	// For service, right now, only Fqdn can be patched.
	// Ensure that we are passed a valid Fqdn
	if len(s.Fqdn) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide fqdn using Fqdn field")
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
		return http.StatusBadRequest, "Please provide service name using Name field"
	}

	if len(s.Fqdn) == 0 {
		return http.StatusBadRequest, "Please provide fqdn using Fqdn field"
	}

	return http.StatusOK, ""
}

func POST_Service_Copy(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name_src := c.Param("service_name_src")
	service_name_dst := c.Param("service_name_dst")

	code, buf := db_copy_service(service_name_src, service_name_dst, log)

	return c.JSONBlob(code, []byte(buf))
}

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

func GET_Service(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	code, buf := db_get_service(log)
	return c.JSONBlob(code, []byte(buf))
}

func GET_One_Service(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	code, buf, _ := db_get_one_service(service_name, false, log)
	return c.JSONBlob(code, []byte(buf))
}

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

func db_insert_service_route(service_name string, r *Route, log *logrus.Entry) (int, string) {

	var QPostServiceRoute = `
mutation insert_service_route($route_name: String!, $route_prefix: String!, $service_name: String!) {
  insert_saaras_db_route(
    objects: 
    {
      route_name: $route_name, 
      route_prefix: $route_prefix, 
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
	var args map[string]string
	args = make(map[string]string)
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["route_name"] = r.Route_name
	args["route_prefix"] = r.Route_prefix
	args["service_name"] = service_name

	if err := saaras.RunDBQuery(url, QPostServiceRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	return http.StatusCreated, buf.String()
}

func validate_service_route(r *Route) (int, string) {
	if len(r.Route_name) == 0 {
		return http.StatusBadRequest, "Please provide route name using Name field"
	}

	if len(r.Route_prefix) == 0 {
		return http.StatusBadRequest, "Please provide route prefix using Prefix field"
	}

	return http.StatusOK, ""
}

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
	return c.JSONBlob(http.StatusOK, buf.Bytes())
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

func POST_Service_Route_Upstream_Associate(c echo.Context) error {
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

	if err := saaras.RunDBQuery(url, QAssociateRouteUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

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
	e.POST("/service/copy/:service_name_src/:service_name_dst/route/:route_name", POST_Service_Route_Copy)
}
