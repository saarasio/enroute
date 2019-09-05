package webhttp

import (
	"bytes"
	"github.com/labstack/echo"
	"github.com/saarasio/enroute/saaras"
	"net/http"

	"github.com/sirupsen/logrus"
)

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

func PATCH_Service(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Service)
	if err := c.Bind(s); err != nil {
		return err
	}

	service_name := c.Param("service_name")

	if len(s.Fqdn) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide fqdn using Fqdn field")
	}

	args["service_name"] = service_name
	args["fqdn"] = s.Fqdn
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.FetchConfig2(url, QPatchService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
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

	if len(s.Service_name) == 0 {
		return http.StatusBadRequest, "Please provide service name using Name field"
	}

	if len(s.Fqdn) == 0 {
		return http.StatusBadRequest, "Please provide fqdn using Fqdn field"
	}

	args["service_name"] = s.Service_name
	args["fqdn"] = s.Fqdn

	if err := saaras.FetchConfig2(url, QCreateService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return http.StatusCreated, buf.String()
}

func POST_Service(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Service)
	if err := c.Bind(s); err != nil {
		return err
	}

	code, buf := db_insert_service(s, log)
	return c.JSONBlob(code, []byte(buf))
}

func GET_Service(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.FetchConfig2(url, QGetService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
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
	if err := saaras.FetchConfig2(url, QDeleteService, &buf, args, log); err != nil {
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
	if err := saaras.FetchConfig2(url, QServiceProxy, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func POST_Service_Route(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	r := new(Route)
	if err := c.Bind(r); err != nil {
		return err
	}

	service_name := c.Param("service_name")
	args["service_name"] = service_name

	if len(r.Route_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide route name using Name field")
	}

	if len(r.Route_prefix) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide route prefix using Prefix field")
	}

	args["route_name"] = r.Route_name
	args["route_prefix"] = r.Route_prefix

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.FetchConfig2(url, QPostServiceRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
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

	if err := saaras.FetchConfig2(url, QServiceRoutes, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())

}

func GET_Service_Route_OneRoute(c echo.Context) error {
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

	if err := saaras.FetchConfig2(url, QServiceRouteOneRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())

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

	if err := saaras.FetchConfig2(url, QDeleteRouteOneRoute, &buf, args, log); err != nil {
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

	if err := saaras.FetchConfig2(url, QAssociateRouteUpstream, &buf, args, log); err != nil {
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

	if err := saaras.FetchConfig2(url, QGetServiceRouteUpstreams, &buf, args, log); err != nil {
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

	if err := saaras.FetchConfig2(url, QDeleteServiceRouteUpstreamAssociation, &buf, args, log); err != nil {
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

	if err := saaras.FetchConfig2(url, QGetServiceSecret, &buf, args, log); err != nil {
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

	if err := saaras.FetchConfig2(url, QAssociateServiceSecret, &buf, args, log); err != nil {
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

	if err := saaras.FetchConfig2(url, QDisassociateServiceSecret, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func Add_service_routes(e *echo.Echo) {

	// Service CRUD
	e.GET("/service", GET_Service)
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
}
