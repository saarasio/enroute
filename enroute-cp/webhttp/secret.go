// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package webhttp

import (
	"bytes"
	"github.com/labstack/echo/v4"
	"github.com/saarasio/enroute/enroute-dp/saaras"
	"net/http"

	"github.com/sirupsen/logrus"
)

var QCreateSecret = `
mutation insert_secret($secret_name:String!, $secret_key:String!, $secret_cert:String, $secret_sni:String)
{
  insert_saaras_db_secret(
    objects: 
    {
      secret_name: $secret_name, 
      secret_key: $secret_key, 
      secret_cert: $secret_cert, 
      secret_sni: $secret_sni
    }
  ) 
  {
    affected_rows
  }
}
`

var QCreateSecretKey = `
mutation insert_secret($secret_name: String!, $secret_key: String!) {
  update_saaras_db_secret(
    where: {secret_name: {_eq: $secret_name}}, 
    _set: {secret_key: $secret_key}) {
    affected_rows
  }
}
`

var QGetSecret = `
query get_secret {
  saaras_db_secret {
    secret_name
    secret_key
    secret_cert
    secret_sni
    create_ts
    update_ts
  }
}
`

var QUpdateSecretKey = `
mutation update_secret_key($secret_name: String!, $secret_key: String!){
  update_saaras_db_secret(
    where: 
    {
      secret_name: {_eq: $secret_name}
    }, 
    _set: 
    {
      secret_key: $secret_key
    }
  ) 
  {
    affected_rows
  }
}
`

var QUpdateSecretCert = `
mutation update_secret_key($secret_name: String!, $secret_cert: String!){
  update_saaras_db_secret(
    where: 
    {
      secret_name: {_eq: $secret_name}
    }, 
    _set: 
    {
      secret_cert: $secret_cert
    }
  ) 
  {
    affected_rows
  }
}
`

var QDeleteSecret = `
mutation delete_secret($secret_name: String!){
        delete_saaras_db_secret(where: {secret_name: {_eq: $secret_name}}) {
                affected_rows
        }
}
`

// @Summary Create a secret
// @Tags secret
// @Accept  json
// @Produce  json
// @Param Secret body webhttp.Secret true "Secret to create"
// @Success 200 {number} uint OK
// @Router /secret [post]
// @Security ApiKeyAuth
func POST_Secret(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	s := new(Secret)
	if err := c.Bind(s); err != nil {
		return err
	}

	if len(s.Secret_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide secret name")
	}

	args["secret_name"] = s.Secret_name
	args["secret_key"] = s.Secret_key
	args["secret_cert"] = s.Secret_cert
	args["secret_sni"] = s.Secret_sni

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QCreateSecret, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// POST key from a file
// curl -X POST -F 'Secret_key=@private_key.pem' http://localhost:1323/secret/testsecret/key | python -m json.tool

// @Summary Set the secret key from file
// @Description Set the secret key from file
// @Description Example curl -X POST -F 'Secret_key=@private_key.pem' http://localhost:1323/secret/testsecret/key | python -m json.tool
// @Tags secret
// @Accept  json
// @Produce  json
// @Param secret_name path string true "Name of secret"
// @Param secret_key formData file true "Location of file holding the secret key"
// @Success 200 {number} uint OK
// @Router /secret/{secret_name}/key [post]
// @Security ApiKeyAuth
func POST_Secret_Key(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	file, err := c.FormFile("Secret_key")
	if file == nil {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Secret_key empty\"}")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	buf2 := new(bytes.Buffer)
	buf2.ReadFrom(src)
	secret_key := buf2.String()

	secret_name := c.Param("secret_name")

	if len(secret_name) == 0 {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Please provide secret name\"}")
	}

	args["secret_name"] = secret_name
	args["secret_key"] = secret_key

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QUpdateSecretKey, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// POST key from a file
// curl -X POST -F 'Secret_cert=@certificate.pem' http://localhost:1323/secret/testsecret/cert | python -m json.tool

// @Summary Set the secret cert from file
// @Description Set the secret cert from file
// @Description Example curl -X POST -F 'Secret_cert=@certificate.pem' http://localhost:1323/secret/testsecret/cert | python -m json.tool
// @Tags secret
// @Accept  json
// @Produce  json
// @Param secret_name path string true "Name of secret"
// @Param secret_cert formData file true "Location of file holding the secret cert"
// @Success 200 {number} uint OK
// @Router /secret/{secret_name}/cert [post]
// @Security ApiKeyAuth
func POST_Secret_Cert(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	file, err := c.FormFile("Secret_cert")
	if file == nil {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Secret_cert empty\"}")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	buf2 := new(bytes.Buffer)
	buf2.ReadFrom(src)
	secret_cert := buf2.String()

	secret_name := c.Param("secret_name")

	if len(secret_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide secret name")
	}

	args["secret_name"] = secret_name
	args["secret_cert"] = secret_cert

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QUpdateSecretCert, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary List all secrets
// @Description Get a list of all secrets for all services
// @Tags secret
// @Accept  json
// @Produce  json
// @Success 200 {number} uint OK
// @Router /secret [get]
// @Security ApiKeyAuth
func GET_Secret(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QGetSecret, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Delete a secret
// @Tags secret
// @Accept  json
// @Produce  json
// @Param secret_name path string true "Name of secret"
// @Success 200 {number} uint OK
// @Router /secret/{secret_name} [delete]
// @Security ApiKeyAuth
func DELETE_Secret(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	secret_name := c.Param("secret_name")
	args["secret_name"] = secret_name
	if err := saaras.RunDBQuery(url, QDeleteSecret, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func Add_secret_routes(e *echo.Echo) {

	// Upstream CRUD
	e.GET("/secret", GET_Secret)
	e.POST("/secret", POST_Secret)
	e.POST("/secret/:secret_name/key", POST_Secret_Key)
	e.POST("/secret/:secret_name/cert", POST_Secret_Cert)
	//	e.POST("/secret/:secret_name/sni", POST_Secret_SNI)
	e.DELETE("/secret/:secret_name", DELETE_Secret)
}
