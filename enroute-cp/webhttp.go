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

var SECRET = "treehugger"

// @title Enroute API
// @version 1.0
// @description APIs to configure multiple envoy proxies
// @contact.name API Support
// @contact.url https://saaras.io/
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	e := echo.New()

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
			return key == SECRET, nil
		},
	}

	e.Use(middleware.KeyAuthWithConfig(config))
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.Use(middleware.Logger())

	e.HideBanner = true
	webhttp.Add_proxy_routes(e)
	webhttp.Add_service_routes(e)
	webhttp.Add_upstream_routes(e)
	webhttp.Add_secret_routes(e)
	webhttp.HOST = os.Getenv("DB_HOST")
	webhttp.PORT = os.Getenv("DB_PORT")

	if webhttp.HOST == "" {
		webhttp.HOST = "127.0.0.1"
	}
	if webhttp.PORT == "" {
		webhttp.PORT = "8080"
	}

	fmt.Printf(" DB_HOST set to [%s] DB_PORT set to [%s]\n", webhttp.HOST, webhttp.PORT)
	e.Logger.Fatal(e.Start("0.0.0.0:1323"))
}
