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
	if err := saaras.FetchConfig2(url, QGetProxy, &buf, args, log); err != nil {
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

func Add_endpoint_service(e *echo.Echo) {
	//e.GET("/service", GET_Service)
	e.PATCH("/service/:service_name", PATCH_Service)
	//e.DELETE("/service", DELETE_Service)
}
