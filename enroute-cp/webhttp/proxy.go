package webhttp

import (
	"bytes"
	"github.com/labstack/echo"
	"github.com/saarasio/enroute/saaras"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Proxy struct {
	Name string `json:"name" xml:"name" form:"name" query:"name"`
}

type Service struct {
	Service_name string `json:"service_name" xml:"service_name" form:"service_name" query:"service_name"`
	Fqdn         string `json:"fqdn" xml:"fqdn" form:"fqdn" query:"fqdn"`
}

type Route struct {
	Route_name   string `json:"route_name" xml:"route_name" form:"route_name" query:"route_name"`
	Route_prefix string `json:"route_prefix" xml:"route_prefix" form:"route_prefix" query:"route_prefix"`
}

type Upstream struct {
	Upstream_name    string `json:"upstream_name" xml:"upstream_name" form:"upstream_name" query:"upstream_name"`
	Upstream_ip      string `json:"upstream_ip" xml:"upstream_ip" form:"upstream_ip" query:"upstream_ip"`
	Upstream_port    string `json:"upstream_port" xml:"upstream_port" form:"upstream_port" query:"upstream_port"`
	Upstream_hc_path string `json:"upstream_hc_path" xml:"upstream_hc_path" form:"upstream_hc_path" query:"upstream_hc_path"`
	Upstream_hc_host string `json:"upstream_hc_host" xml:"upstream_hc_host" form:"upstream_hc_host" query:"upstream_hc_host"`
}

type Secret struct {
	Secret_name string `json:"secret_name" xml:"secret_name" form:"secret_name" query:"secret_name"`
	Secret_key  string `json:"secret_key" xml:"secret_key" form:"secret_key" query:"secret_key"`
	Secret_cert string `json:"secret_cert" xml:"secret_cert" form:"secret_cert" query:"secret_cert"`
	Secret_sni  string `json:"secret_sni" xml:"secret_sni" form:"secret_sni" query:"secret_sni"`
}

var QCreateProxy string = `
    mutation create_proxy($proxy_name : String!){
      insert_saaras_db_proxy(objects: {proxy_name: $proxy_name},
        on_conflict: {constraint: proxy_proxy_name_key, update_columns: create_ts}) {
        affected_rows
      }
    }
`

var QGetProxy string = `
query get_proxies {
  saaras_db_proxy {
    proxy_id
    proxy_name
    create_ts
    update_ts
  }
}
`

var QDeleteProxy string = `
mutation delete_proxy($proxy_name: String!) {
  delete_saaras_db_proxy(where: {proxy_name: {_eq: $proxy_name}}) {
    affected_rows
  }
}
`

var QCreateProxyService string = `
      mutation create_proxy_service($proxy_name : String!, $fqdn : String!, $service_name : String!) {
				# Create service
        insert_saaras_db_service
        (
          objects:
          {
            fqdn: $fqdn,
            service_name: $service_name
          } on_conflict: {constraint: service_service_name_key, update_columns:[fqdn, service_name]}
        )
        {
          returning
          {
            create_ts
          }
        }

        # Associate a service to a proxy
        insert_saaras_db_proxy_service(
          objects:
          {
            proxy:
            {
              data:
              {
                proxy_name: $proxy_name
              }, on_conflict: {constraint: proxy_proxy_name_key, update_columns: update_ts}
            },
            service:
            {
              data:
              {
                service_name: $service_name
              }, on_conflict: {constraint: service_service_name_key, update_columns: update_ts}
            }
          }
        )
        {
          affected_rows
        }
      }
`

var QGetProxyService string = `
	query get_proxy_service($proxy_name: String!) {
	  saaras_db_service(where: {proxy_services: {proxy: {proxy_name: {_eq: $proxy_name}}}}) {
	    service_id
	    service_name
	    fqdn
	    create_ts
	    update_ts
	  }
	}
`

var QGetProxyServiceAssociation string = `
query get_proxy_service($proxy_name: String!, $service_name: String!) {
  saaras_db_service(where: {_and: 
    {proxy_services: {proxy: {proxy_name: {_eq: $proxy_name}}}}, service_name: {_eq: $service_name}}) {
    service_id
    service_name
    fqdn
    create_ts
    update_ts
  }
}
`

var QDeleteProxyService = `

mutation delete_proxy_service($service_name: String!, $proxy_name: String!) {
  delete_saaras_db_proxy_service(where: {
		_and:
			{
				proxy: {proxy_name: {_eq: $proxy_name}}, 
				service: {service_name: {_eq: $service_name}}
			}
		}) {
    affected_rows
  }
  delete_saaras_db_service(where: {service_name: {_eq: $service_name}}) {
    affected_rows
  }
}

`

var QDeleteProxyServiceAssociation = `

mutation delete_proxy_service($service_name: String!, $proxy_name: String!) {
  delete_saaras_db_proxy_service(where: {
		_and:
			{
				proxy: {proxy_name: {_eq: $proxy_name}}, 
				service: {service_name: {_eq: $service_name}}
			}
		}) {
    affected_rows
  }
}
`

var QCreateProxyServiceAssociation = `
      mutation create_proxy_service($proxy_name : String!, $service_name : String!) {
        # Associate a service to a proxy
        insert_saaras_db_proxy_service(
          objects:
          {
            proxy:
            {
              data:
              {
                proxy_name: $proxy_name
              }, on_conflict: {constraint: proxy_proxy_name_key, update_columns: update_ts}
            },
            service:
            {
              data:
              {
                service_name: $service_name
              }, on_conflict: {constraint: service_service_name_key, update_columns: update_ts}
            }
          }
        )
        {
          affected_rows
        }
      }
`

var QGetAllProxyDetail = `
query get_proxy_detail {
  saaras_db_proxy {
    proxy_id
    proxy_name
    create_ts
    update_ts
    proxy_services {
      service {
        service_id
        service_name
        fqdn
        create_ts
        update_ts
        service_secrets {
          secret {
            secret_id
            secret_name
            secret_key
            secret_cert
            create_ts
            update_ts
          }
        }
        routes {
          route_id
          route_name
          route_prefix
          create_ts
          update_ts
          route_upstreams {
            upstream {
              upstream_id
              upstream_name
              upstream_ip
              upstream_port
              upstream_hc_healthythresholdcount
              upstream_hc_host
              upstream_hc_intervalseconds
              upstream_hc_path
              upstream_hc_timeoutseconds
              upstream_hc_unhealthythresholdcount
              upstream_strategy
              upstream_validation_cacertificate
              upstream_validation_subjectname
              upstream_weight
            }
          }
        }
      }
    }
  }
}
`

var QGetOneProxyDetail = `
query get_one_proxy_detail($proxy_name:String!) {
  saaras_db_proxy(where: {proxy_name: {_eq: $proxy_name}}) {
    proxy_id
    proxy_name
    create_ts
    update_ts
    proxy_services {
      service {
        service_id
        service_name
        fqdn
        create_ts
        update_ts
        service_secrets {
          secret {
            secret_id
            secret_name
            secret_key
            secret_cert
            create_ts
            update_ts
          }
        }
        routes {
          route_id
          route_name
          route_prefix
          create_ts
          update_ts
          route_upstreams {
            upstream {
              upstream_id
              upstream_name
              upstream_ip
              upstream_port
              upstream_hc_healthythresholdcount
              upstream_hc_host
              upstream_hc_intervalseconds
              upstream_hc_path
              upstream_hc_timeoutseconds
              upstream_hc_unhealthythresholdcount
              upstream_strategy
              upstream_validation_cacertificate
              upstream_validation_subjectname
              upstream_weight
            }
          }
        }
      }
    }
  }
}
`

var HOST string = `localhost`
var PORT string = `8081`

func POST_Proxy(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	p := new(Proxy)
	if err := c.Bind(p); err != nil {
		return err
	}

	if len(p.Name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide name of proxy using Name field")
	}

	args["proxy_name"] = p.Name
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QCreateProxy, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSON(http.StatusCreated, p)
}

func POST_Proxy_Service(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Service)
	if err := c.Bind(s); err != nil {
		return err
	}

	if len(s.Service_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide name of proxy using Name field")
	}

	if len(s.Fqdn) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide fqdn using Fqdn field")
	}

	proxy_name := c.Param("proxy_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["proxy_name"] = proxy_name
	args["fqdn"] = s.Fqdn
	args["service_name"] = s.Service_name

	if err := saaras.RunDBQuery(url, QCreateProxyService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

func GET_Proxy(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QGetProxy, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_Proxy_Detail(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QGetAllProxyDetail, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_One_Proxy_Detail(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	proxy_name := c.Param("proxy_name")
	args["proxy_name"] = proxy_name

	if err := saaras.RunDBQuery(url, QGetOneProxyDetail, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_Proxy_Service(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")
	proxy_name := c.Param("proxy_name")

	args["proxy_name"] = proxy_name
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QGetProxyService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func DELETE_Proxy(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	p := new(Proxy)
	if err := c.Bind(p); err != nil {
		return err
	}

	if len(p.Name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide name of proxy using Name field")
	}

	args["proxy_name"] = p.Name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QDeleteProxy, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())

}

func DELETE_Proxy_Service(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Service)
	if err := c.Bind(s); err != nil {
		return err
	}

	if len(s.Service_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide name of proxy using Name field")
	}

	proxy_name := c.Param("proxy_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["proxy_name"] = proxy_name
	args["service_name"] = s.Service_name

	if err := saaras.RunDBQuery(url, QDeleteProxyService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func POST_Proxy_Service_Association(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	proxy_name := c.Param("proxy_name")
	service_name := c.Param("service_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["proxy_name"] = proxy_name
	args["service_name"] = service_name

	if err := saaras.RunDBQuery(url, QCreateProxyServiceAssociation, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

func DELETE_Proxy_Service_Association(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	proxy_name := c.Param("proxy_name")
	service_name := c.Param("service_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["proxy_name"] = proxy_name
	args["service_name"] = service_name

	if err := saaras.RunDBQuery(url, QDeleteProxyServiceAssociation, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_Proxy_Service_Association(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")
	proxy_name := c.Param("proxy_name")
	service_name := c.Param("service_name")

	args["proxy_name"] = proxy_name
	args["service_name"] = service_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QGetProxyServiceAssociation, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func Add_proxy_routes(e *echo.Echo) {
	// Proxy CRUD
	e.GET("/proxy", GET_Proxy)
	e.POST("/proxy", POST_Proxy)
	e.DELETE("/proxy", DELETE_Proxy)

	// Proxy to Service association with implied service CRUD
	// Only the GET makes sense here?
	// e.POST("/proxy/:proxy_name/service", POST_Proxy_Service)
	e.GET("/proxy/:proxy_name/service", GET_Proxy_Service)
	// e.DELETE("/proxy/:proxy_name/service", DELETE_Proxy_Service)

	// Proxy to Service association
	e.POST("/proxy/:proxy_name/service/:service_name", POST_Proxy_Service_Association)
	e.GET("/proxy/:proxy_name/service/:service_name", GET_Proxy_Service_Association)
	e.DELETE("/proxy/:proxy_name/service/:service_name", DELETE_Proxy_Service_Association)

	// Support for verbs
	e.GET("/proxy/dump", GET_Proxy_Detail)
	e.GET("/proxy/dump/:proxy_name", GET_One_Proxy_Detail)
}
