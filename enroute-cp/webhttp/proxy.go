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
	Name string `json:"name" xml:"name" form:"name" query:"name"`
	Fqdn string `json:"fqdn" xml:"fqdn" form:"fqdn" query:"fqdn"`
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

	if err := saaras.FetchConfig2(url, QCreateProxy, &buf, args, log); err != nil {
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

	if len(s.Name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide name of proxy using Name field")
	}

	if len(s.Fqdn) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide fqdn using Fqdn field")
	}

	proxy_name := c.Param("proxy_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["proxy_name"] = proxy_name
	args["fqdn"] = s.Fqdn
	args["service_name"] = s.Name

	if err := saaras.FetchConfig2(url, QCreateProxyService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_Proxy(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.FetchConfig2(url, QGetProxy, &buf, args, log); err != nil {
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
	if err := saaras.FetchConfig2(url, QGetProxyService, &buf, args, log); err != nil {
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
	if err := saaras.FetchConfig2(url, QDeleteProxy, &buf, args, log); err != nil {
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

	if len(s.Name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide name of proxy using Name field")
	}

	proxy_name := c.Param("proxy_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["proxy_name"] = proxy_name
	args["service_name"] = s.Name

	if err := saaras.FetchConfig2(url, QDeleteProxyService, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func Add_endpoint_proxy(e *echo.Echo) {
	e.GET("/proxy", GET_Proxy)
	e.POST("/proxy", POST_Proxy)
	e.DELETE("/proxy", DELETE_Proxy)

	e.POST("/proxy/:proxy_name/service", POST_Proxy_Service)
	e.GET("/proxy/:proxy_name/service", GET_Proxy_Service)
	e.DELETE("/proxy/:proxy_name/service", DELETE_Proxy_Service)
}
