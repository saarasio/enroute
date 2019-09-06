package webhttp

import (
	"bytes"
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/saarasio/enroute/saaras"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// TODO: Build query to only patch fields that need updating
var QPatchUpstream = `
mutation update_upstream($upstream_name: String!, $upstream_ip: String!, $upstream_port: Int!, $upstream_hc_path: String!) {
  update_saaras_db_upstream(
    where: {upstream_name: {_eq: $upstream_name}}, 
    _set: {
			upstream_ip: $upstream_ip, 
			upstream_port: $upstream_port, 
			upstream_hc_path: $upstream_hc_path
		}
	) {
    affected_rows
  }
}
`

var QGetUpstream = `
query get_upstream {
  saaras_db_upstream {
    upstream_id
    upstream_name
    upstream_ip
    upstream_port
    upstream_hc_path
    upstream_hc_host
    upstream_hc_intervalseconds
    upstream_hc_timeoutseconds
    upstream_hc_unhealthythresholdcount
    upstream_hc_healthythresholdcount
    upstream_strategy
    upstream_validation_cacertificate
    upstream_validation_subjectname
    create_ts
    update_ts
  }
}
`

var QDeleteUpstream = `
mutation delete_upstream($upstream_name: String!) {
  delete_saaras_db_upstream(where: {upstream_name: {_eq: $upstream_name}}) {
    affected_rows
  }
}
`

var QUpstreamRoutes = `
query get_upstream_routes($upstream_name: String!) {
  saaras_db_upstream(where: {upstream_name: {_eq: $upstream_name}}) {
    route_upstreams {
      route {
        route_id
        route_name
        route_prefix
        create_ts
        update_ts
      }
    }
  }
}
`

func PATCH_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	u := new(Upstream)
	if err := c.Bind(u); err != nil {
		return err
	}

	if len(u.Upstream_name) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide upstream name using Name field")
	}

	if len(u.Upstream_ip) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide Ip using Ip field")
	}

	if len(u.Upstream_port) == 0 {
		return c.JSON(http.StatusBadRequest, "Please provide Port using Port field")
	}

	args["upstream_name"] = u.Upstream_name
	args["upstream_ip"] = u.Upstream_ip
	args["upstream_port"] = u.Upstream_port

	if len(u.Upstream_hc_path) > 0 {
		args["upstream_hc_path"] = u.Upstream_hc_path
	}

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QPatchUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func db_insert_upstream(u *Upstream, log *logrus.Entry) (int, string) {
	var QCreateUpstream = `
	mutation insert_upstream(
		$upstream_name: String!, 
		$upstream_ip: String!, 
		$upstream_port: Int!,
		$upstream_hc_path: String!,
		$upstream_hc_host: String!
	) {
	  insert_saaras_db_upstream(
	    objects: {
	      upstream_name: $upstream_name, 
	      upstream_ip: $upstream_ip, 
	      upstream_port: $upstream_port,
	      upstream_hc_path: $upstream_hc_path,
	      upstream_hc_host: $upstream_hc_host
	    }
	  ) {
	    affected_rows
	  }
	}
	`

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	args["upstream_name"] = u.Upstream_name
	args["upstream_ip"] = u.Upstream_ip
	args["upstream_port"] = u.Upstream_port
	args["upstream_hc_path"] = u.Upstream_hc_path
	args["upstream_hc_host"] = u.Upstream_hc_host

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QCreateUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	return http.StatusCreated, buf.String()
}

func validate_upstream(u *Upstream) (int, string) {
	if len(u.Upstream_name) == 0 {
		return http.StatusBadRequest, "Please provide upstream name using Name field"
	}

	if len(u.Upstream_ip) == 0 {
		return http.StatusBadRequest, "Please provide Ip using Ip field"
	}

	if len(u.Upstream_port) == 0 {
		return http.StatusBadRequest, "Please provide Port using Port field"
	}
	// TODO: Should we make health check path mandatory? Without the path, the health checker is
	// is not programmed and it is not getting programmed on envoy through CDS/EDS
	if len(u.Upstream_hc_path) > 0 {
	} else {
		return http.StatusBadRequest, "Please provide a value for upstream_hc_path"
	}

	if len(u.Upstream_hc_host) > 0 {
	} else {
		return http.StatusBadRequest, "Please provide a value for upstream_hc_host"
	}

	return http.StatusOK, ""
}

// TODO: Add support for the following fields -
//
//            upstream_hc_intervalseconds
//            upstream_hc_timeoutseconds
//            upstream_hc_unhealthythresholdcount
//            upstream_hc_healthythresholdcount
//            upstream_strategy
//            upstream_validation_cacertificate
//            upstream_validation_subjectname

func POST_Upstream(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	u := new(Upstream)
	if err := c.Bind(u); err != nil {
		return err
	}

	code, buf := validate_upstream(u)

	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	code2, buf2 := db_insert_upstream(u, log)
	return c.JSONBlob(code2, []byte(buf2))
}

func db_get_one_upstream(upstream_name string, decode bool, log *logrus.Entry) (int, string, *Upstream) {
	// TODO: Capture all fields in upstream
	var QOneUpstream = `
	query get_upstream($upstream_name : String!) {
		saaras_db_upstream(where: {upstream_name: {_eq: $upstream_name}}) {
			upstream_id
			upstream_name
			upstream_ip
			upstream_port
                        upstream_hc_path
                        upstream_hc_host
			create_ts
			update_ts
		}
	}
	`

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)
	args["upstream_name"] = upstream_name
	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	if err := saaras.RunDBQuery(url, QOneUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	// {
	//   "data": {
	//     "saaras_db_upstream": [
	//       {
	//         "upstream_id": 3,
	//         "upstream_name": "test",
	//         "upstream_ip": "127.0.0.1",
	//         "upstream_port": 9001,
	//         "create_ts": "2019-09-05T02:05:01.31547+00:00",
	//         "update_ts": "2019-09-05T03:13:31.854692+00:00"
	//       }
	//     ]
	//   }
	// }

	type SaarasDbUpstream struct {
		UpstreamID     int       `json:"upstream_id"`
		UpstreamName   string    `json:"upstream_name"`
		UpstreamIP     string    `json:"upstream_ip"`
		UpstreamPort   int       `json:"upstream_port"`
		UpstreamHcPath string    `json:"upstream_hc_path"`
		UpstreamHcHost string    `json:"upstream_hc_host"`
		CreateTs       time.Time `json:"create_ts"`
		UpdateTs       time.Time `json:"update_ts"`
	}
	type Data struct {
		SaarasDbUpstream []SaarasDbUpstream `json:"saaras_db_upstream"`
	}
	type AutoGenerated struct {
		Data Data `json:"data"`
	}

	var u Upstream

	if decode {
		var gr AutoGenerated
		log.Infof("Decoding :\n %s\n", buf.String())
		if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
			errors.Wrap(err, "decoding response")
			log.Errorf("Error when decoding json [%v]\n", err)
			return http.StatusBadRequest, buf.String(), nil
		}

		if len(gr.Data.SaarasDbUpstream) > 0 {
			u.Upstream_name = gr.Data.SaarasDbUpstream[0].UpstreamName
			u.Upstream_ip = gr.Data.SaarasDbUpstream[0].UpstreamIP
			u.Upstream_port = strconv.FormatInt(int64(gr.Data.SaarasDbUpstream[0].UpstreamPort), 10)
			u.Upstream_hc_path = gr.Data.SaarasDbUpstream[0].UpstreamHcPath
			u.Upstream_hc_host = gr.Data.SaarasDbUpstream[0].UpstreamHcHost
		}

	}

	log.Infof("Decoded to upstream [%v]\n", u)

	return http.StatusOK, buf.String(), &u
}

func GET_One_Upstream(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	upstream_name := c.Param("upstream_name")
	code, buf, _ := db_get_one_upstream(upstream_name, false, log)
	return c.JSONBlob(code, []byte(buf))
}

func POST_Upstream_Copy(c echo.Context) error {
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	upstream_name_src := c.Param("upstream_name_src")
	upstream_name_dst := c.Param("upstream_name_dst")
	code, buf, u := db_get_one_upstream(upstream_name_src, true, log)

	if code != http.StatusOK {
		return c.JSONBlob(code, []byte(buf))
	}

	u.Upstream_name = upstream_name_dst
	code2, buf2 := db_insert_upstream(u, log)
	return c.JSONBlob(code2, []byte(buf2))

}

func GET_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, QGetUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func DELETE_Upstream(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	upstream_name := c.Param("upstream_name")
	args["upstream_name"] = upstream_name
	if err := saaras.RunDBQuery(url, QDeleteUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func GET_Upstream_Routes(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	upstream_name := c.Param("upstream_name")
	args["upstream_name"] = upstream_name
	if err := saaras.RunDBQuery(url, QUpstreamRoutes, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func Add_upstream_routes(e *echo.Echo) {

	// Upstream CRUD
	e.GET("/upstream", GET_Upstream)
	e.POST("/upstream", POST_Upstream)
	e.PATCH("/upstream/:upstream_name", PATCH_Upstream)
	e.DELETE("/upstream/:upstream_name", DELETE_Upstream)
	e.GET("/upstream/:upstream_name", GET_One_Upstream)

	// Get all routes associated with this upstream
	e.GET("/upstream/:upstream_name/route", GET_Upstream_Routes)

	// Support for verbs
	e.POST("/upstream/copy/:upstream_name_src/:upstream_name_dst", POST_Upstream_Copy)
}
