package main

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	webhttp "github.com/saarasio/enroute/enroute-cp/webhttp"
	"os"
)

func main() {
	e := echo.New()
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
