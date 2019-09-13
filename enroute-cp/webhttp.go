package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	webhttp "github.com/saarasio/enroute/enroute-cp/webhttp"
	"os"
	"github.com/swaggo/echo-swagger"
	_ "github.com/saarasio/enroute/enroute-cp/docs"
)

// @title Enroute API
// @version 1.0
// @description APIs to configure multiple envoy proxies

// @contact.name API Support
// @contact.url https://saaras.io/

func main() {
	e := echo.New()
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.HideBanner = true
	webhttp.Add_proxy_routes(e)
	webhttp.Add_service_routes(e)
	webhttp.Add_upstream_routes(e)
	webhttp.Add_secret_routes(e)
	e.Use(middleware.Logger())
	webhttp.HOST = os.Getenv("DB_HOST")
	webhttp.PORT = os.Getenv("DB_PORT")

	if webhttp.HOST == "" {
		webhttp.HOST = "127.0.0.1"
	}
	if webhttp.PORT == "" {
		webhttp.PORT = "8081"
	}
	fmt.Printf(" DB_HOST set to [%s] DB_PORT set to [%s]\n", webhttp.HOST, webhttp.PORT)
	e.Logger.Fatal(e.Start("127.0.0.1:1323"))
}
