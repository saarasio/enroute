module github.com/saarasio/enroute/enroute-cp

go 1.12

require (
	github.com/labstack/echo/v4 v4.1.10
	github.com/pkg/errors v0.8.1
	github.com/saarasio/enroute/enroute-cp/docs v0.1.0
	github.com/saarasio/enroute/enroute-dp v0.1.0
	github.com/sirupsen/logrus v1.4.2
	github.com/swaggo/echo-swagger v0.0.0-20190329130007-1219b460a043
)

replace github.com/saarasio/enroute/enroute-cp/docs => ./docs

replace github.com/saarasio/enroute/enroute-dp => ../enroute-dp
