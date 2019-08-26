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

func Add_endpoint_proxy(e *echo.Echo) {
	e.GET("/proxy", GET_Proxy)
	e.POST("/proxy", POST_Proxy)
	e.DELETE("/proxy", DELETE_Proxy)
}


