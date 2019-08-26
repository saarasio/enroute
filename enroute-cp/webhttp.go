package main

import (
	"github.com/labstack/echo"
	webhttp "github.com/saarasio/enroute/enroute-cp/webhttp"
)

func main() {
	e := echo.New()
	webhttp.Add_endpoint_proxy(e)
	e.Logger.Fatal(e.Start(":1323"))
}
