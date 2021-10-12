module github.com/saarasio/enroute/enroute-cp

go 1.15

require (
	github.com/labstack/echo/v4 v4.6.1
	github.com/pkg/errors v0.9.1
	github.com/saarasio/enroute/enroute-cp/docs v0.7.0
	github.com/saarasio/enroute/enroute-dp v0.7.0
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/swaggo/echo-swagger v1.1.0
)

replace github.com/saarasio/enroute/enroute-cp/docs => ./docs

replace github.com/saarasio/enroute/enroute-dp => ../enroute-dp
