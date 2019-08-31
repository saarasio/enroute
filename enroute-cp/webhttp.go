package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	webhttp "github.com/saarasio/enroute/enroute-cp/webhttp"
)

func main() {
	e := echo.New()
	webhttp.Add_proxy_routes(e)
	webhttp.Add_service_routes(e)
	webhttp.Add_upstream_routes(e)
	webhttp.Add_secret_routes(e)
	e.Use(middleware.Logger())
	e.Logger.Fatal(e.Start(":1323"))
}
