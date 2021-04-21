// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/saarasio/enroute/enroute-cp/docs"
	webhttp "github.com/saarasio/enroute/enroute-cp/webhttp"
	"github.com/swaggo/echo-swagger"
	"os"
	"strings"
)

// @title Enroute API
// @name Enroute Universal API Gateway
// @version 1.0
// @description APIs to configure multiple envoy proxies
// @name Enroute Universal Standalone API Gateway
// @contact.name contact@saaras.io
// @contact.url https://saaras.io/
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
func main() {
	e := echo.New()

	// enviornment
	webhttp.HOST = os.Getenv("DB_HOST")
	webhttp.PORT = os.Getenv("DB_PORT")
	webhttp.SECRET = os.Getenv("WEBAPP_SECRET")

	if webhttp.HOST == "" {
		webhttp.HOST = "127.0.0.1"
	}
	if webhttp.PORT == "" {
		webhttp.PORT = "8080"
	}

	fmt.Printf(" DB_HOST set to [%s] DB_PORT set to [%s] SECRET set to [%s]\n", webhttp.HOST, webhttp.PORT, webhttp.SECRET)

	// middleware

	config := middleware.KeyAuthConfig{
		Skipper: func(c echo.Context) bool {
			uri := c.Request().RequestURI
			if strings.HasPrefix(uri, "/swagger") {
				return true
			}
			return false
		},
		KeyLookup:  "header:" + echo.HeaderAuthorization,
		AuthScheme: "Bearer",
		Validator: func(key string, c echo.Context) (bool, error) {
			return key == webhttp.SECRET, nil
		},
	}

	if webhttp.SECRET != "" {
		e.Use(middleware.KeyAuthWithConfig(config))
	}
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.Use(middleware.Logger())

	e.HideBanner = true
	webhttp.Add_proxy_routes(e)
	webhttp.Add_service_routes(e)
	webhttp.Add_upstream_routes(e)
	webhttp.Add_secret_routes(e)
	webhttp.Add_filter_routes(e)
	go webhttp.Reporter()
	e.Logger.Fatal(e.Start("0.0.0.0:1323"))
}
