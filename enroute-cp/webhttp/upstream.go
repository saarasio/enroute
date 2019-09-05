package webhttp

import (
	"bytes"
	"github.com/labstack/echo"
	"github.com/saarasio/enroute/saaras"
	"net/http"

	"github.com/sirupsen/logrus"
)

// TODO: Build query to only patch fields that need updating
var QPatchUpstream = `
mutation update_upstream($upstream_name: String!, $upstream_ip: String!, $upstream_port: Int!, $upstream_hc_path: String!) {
  update_saaras_db_upstream(
    where: {upstream_name: {_eq: $upstream_name}}, 
    _set: {
			upstream_ip: $upstream_ip, 
			upstream_port: $upstream_port, 
			upstream_hc_path: $upstream_hc_path
		}
	) {
    affected_rows
  }
}
`

var QCreateUpstream = `
mutation insert_upstream(
	$upstream_name: String!, 
	$upstream_ip: String!, 
	$upstream_port: Int!,
	$upstream_hc_path: String!,
	$upstream_hc_host: String!
) {
  insert_saaras_db_upstream(
    objects: {
      upstream_name: $upstream_name, 
      upstream_ip: $upstream_ip, 
      upstream_port: $upstream_port,
			upstream_hc_path: $upstream_hc_path,
			upstream_hc_host: $upstream_hc_host
    }
  ) {
    affected_rows
  }
}
`

var QGetUpstream = `
query get_upstream {
  saaras_db_upstream {
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
`

var QOneUpstream = `
query get_upstream($upstream_name : String!) {
  saaras_db_upstream(where: {upstream_name: {_eq: $upstream_name}}) {
    upstream_id
    upstream_name
    upstream_ip
    upstream_port
    create_ts
    update_ts
  }
}
`

var QDeleteUpstream = `
mutation delete_upstream($upstream_name: String!) {
  delete_saaras_db_upstream(where: {upstream_name: {_eq: $upstream_name}}) {
    affected_rows
  }
}
`

var QUpstreamRoutes = `
query get_upstream_routes($upstream_name: String!) {
  saaras_db_upstream(where: {upstream_name: {_eq: $upstream_name}}) {
    route_upstreams {
      route {
        route_id
        route_name
        route_prefix
        create_ts
        update_ts
      }
    }
  }
}
`

func PATCH_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	u := new(Upstream)
	if err := c.Bind(u); err != nil {
		return err
	}

	if len(u.Upstream_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide upstream name using Name field")
	}

	if len(u.Upstream_ip) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide Ip using Ip field")
	}

	if len(u.Upstream_port) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide Port using Port field")
	}

	args["upstream_name"] = u.Upstream_name
	args["upstream_ip"] = u.Upstream_ip
	args["upstream_port"] = u.Upstream_port

	if len(u.Upstream_hc_path) > 0 {
		args["upstream_hc_path"] = u.Upstream_hc_path
	}

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QPatchUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// TODO: Add support for the following fields -
//
//            upstream_hc_intervalseconds
//            upstream_hc_timeoutseconds
//            upstream_hc_unhealthythresholdcount
//            upstream_hc_healthythresholdcount
//            upstream_strategy
//            upstream_validation_cacertificate
//            upstream_validation_subjectname

func POST_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	u := new(Upstream)
	if err := c.Bind(u); err != nil {
		return err
	}

	if len(u.Upstream_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide upstream name using Name field")
	}

	if len(u.Upstream_ip) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide Ip using Ip field")
	}

	if len(u.Upstream_port) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide Port using Port field")
	}

	args["upstream_name"] = u.Upstream_name
	args["upstream_ip"] = u.Upstream_ip
	args["upstream_port"] = u.Upstream_port

	// TODO: Should we make health check path mandatory? Without the path, the health checker is
	// is not programmed and it is not getting programmed on envoy through CDS/EDS
	if len(u.Upstream_hc_path) > 0 {
		args["upstream_hc_path"] = u.Upstream_hc_path
	} else {
		return c.JSON(http.StatusBadRequest, "Please provide a value for upstream_hc_path")
	}

	if len(u.Upstream_hc_host) > 0 {
		args["upstream_hc_host"] = u.Upstream_hc_host
	} else {
		return c.JSON(http.StatusBadRequest, "Please provide a value for upstream_hc_host")
	}

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QCreateUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_One_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	upstream_name := c.Param("upstream_name")

	args["upstream_name"] = upstream_name
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QOneUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

func GET_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QGetUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func DELETE_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	upstream_name := c.Param("upstream_name")
	args["upstream_name"] = upstream_name
	if err := saaras.RunDBQuery(url, QDeleteUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_Upstream_Routes(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	upstream_name := c.Param("upstream_name")
	args["upstream_name"] = upstream_name
	if err := saaras.RunDBQuery(url, QUpstreamRoutes, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func Add_upstream_routes(e *echo.Echo) {

	// Upstream CRUD
	e.GET("/upstream", GET_Upstream)
	e.POST("/upstream", POST_Upstream)
	e.PATCH("/upstream/:upstream_name", PATCH_Upstream)
	e.DELETE("/upstream/:upstream_name", DELETE_Upstream)
	e.GET("/upstream/:upstream_name", GET_One_Upstream)

	// Get all routes associated with this upstream
	e.GET("/upstram/:upstream_name/route", GET_Upstream_Routes)
}
