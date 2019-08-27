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
query get_proxy_service {
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
mutation insert_service_route($route_name: String!, $service_name: String!) {
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

	if len(r.Name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide route name using Name field")
	}

	if len(r.Prefix) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide route prefix using Prefix field")
	}

	args["route_name"] = r.Name
	args["route_prefix"] = r.Prefix

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.FetchConfig2(url, QPostServiceRoute, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
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

func Add_endpoint_service(e *echo.Echo) {
	e.GET("/service", GET_Service)
	e.GET("/service/:service_name/proxy", GET_Service_Proxy)
	e.PATCH("/service/:service_name", PATCH_Service)
	e.DELETE("/service/:service_name", DELETE_Service)

	e.POST("/service/:service_name/route", POST_Service_Route)
	e.GET("/service/:service_name/route", GET_Service_Route)
	e.GET("/service/:service_name/route/:route_name", GET_Service_Route_OneRoute)
	e.DELETE("/service/:service_name/route/:route_name", DELETE_Service_Route_OneRoute)
}
