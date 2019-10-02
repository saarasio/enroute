package webhttp

import (
	"bytes"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/saarasio/enroute/enroute-dp/saaras"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

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
	 upstream_weight
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

// @Summary Update an upstream
// @Tags upstream
// @Accept  json
// @Produce  json
// @Param Upstream body webhttp.Upstream true "Upstream to update"
// @Param upstream_name path string true "Name of upstream to update"
// @Success 200 {} integer OK
// @Router /upstream/{upstream_name} [patch]
// @Security ApiKeyAuth
func PATCH_Upstream(c echo.Context) error {

	var QPatchUpstream = `

mutation update_upstream(
     $upstream_name: String!,
     $upstream_ip: String!,
     $upstream_port: Int!,
     $upstream_weight: Int!,
     $upstream_hc_path: String!,
     $upstream_hc_host: String!,
     $upstream_hc_intervalseconds: Int!,
     $upstream_hc_unhealthythresholdcount: Int!,
     $upstream_hc_healthythresholdcount: Int!,
     $upstream_strategy: String!,
     $upstream_validation_cacertificate: String!,
     $upstream_validation_subjectname: String!,
     $upstream_hc_timeoutseconds: Int!
 ) {
     update_saaras_db_upstream(
         where: {upstream_name: {_eq: $upstream_name}},
         _set: {
             upstream_ip: $upstream_ip,
             upstream_port: $upstream_port,
             upstream_weight: $upstream_weight,
             upstream_hc_path: $upstream_hc_path,
             upstream_hc_host: $upstream_hc_host,
             upstream_hc_intervalseconds: $upstream_hc_intervalseconds,
             upstream_hc_unhealthythresholdcount: $upstream_hc_unhealthythresholdcount,
             upstream_hc_healthythresholdcount: $upstream_hc_healthythresholdcount,
             upstream_strategy: $upstream_strategy,
             upstream_validation_cacertificate: $upstream_validation_cacertificate,
             upstream_validation_subjectname: $upstream_validation_subjectname,
             upstream_hc_timeoutseconds: $upstream_hc_timeoutseconds
         }
     ) {
         affected_rows
     }
 }
	  `

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	upstream_name := c.Param("upstream_name")

	code2, buf2, u_in_db := db_get_one_upstream(upstream_name, true, log)

	if code2 != http.StatusOK && code2 != http.StatusCreated {
		return c.JSONBlob(code2, []byte(buf2))
	}

	u := new(Upstream)
	if err := c.Bind(u); err != nil {
		return err
	}

	log.Infof(" Read upstream [%+v] \n", u)

	if len(u.Upstream_ip) > 0 {
		u_in_db.Upstream_ip = u.Upstream_ip
	}
	if len(u.Upstream_port) > 0 {
		u_in_db.Upstream_port = u.Upstream_port
	}
	if len(u.Upstream_hc_path) > 0 {
		u_in_db.Upstream_hc_path = u.Upstream_hc_path
	}
	if len(u.Upstream_hc_host) > 0 {
		u_in_db.Upstream_hc_host = u.Upstream_hc_host
	}
	if len(u.Upstream_weight) > 0 {
		u_in_db.Upstream_weight = u.Upstream_weight
	}
	if len(u.Upstream_hc_intervalseconds) > 0 {
		u_in_db.Upstream_hc_intervalseconds = u.Upstream_hc_intervalseconds
	}
	if len(u.Upstream_hc_unhealthythresholdcount) > 0 {
		u_in_db.Upstream_hc_unhealthythresholdcount = u.Upstream_hc_unhealthythresholdcount
	}
	if len(u.Upstream_hc_healthythresholdcount) > 0 {
		u_in_db.Upstream_hc_healthythresholdcount = u.Upstream_hc_healthythresholdcount
	}
	if len(u.Upstream_strategy) > 0 {
		u_in_db.Upstream_strategy = u.Upstream_strategy
	}
	if len(u.Upstream_validation_cacertificate) > 0 {
		u_in_db.Upstream_validation_cacertificate = u.Upstream_validation_cacertificate
	}
	if len(u.Upstream_validation_subjectname) > 0 {
		u_in_db.Upstream_validation_subjectname = u.Upstream_validation_subjectname
	}
	if len(u.Upstream_hc_timeoutseconds) > 0 {
		u_in_db.Upstream_hc_timeoutseconds = u.Upstream_hc_timeoutseconds
	}

	//{
	//  "upstream_name": "test3",
	//  "upstream_ip": "1.1.1.1",
	//  "upstream_port": 11,
	//  "upstream_weight": 11,
	//  "upstream_hc_path": "/",
	//  "upstream_hc_host": "blah",
	//  "upstream_hc_intervalseconds": 11,
	//  "upstream_hc_unhealthythresholdcount": 11,
	//  "upstream_hc_healthythresholdcount": 11,
	//  "upstream_strategy": "blah",
	//  "upstream_validation_cacertificate": "blah",
	//  "upstream_validation_subjectname": "blah",
	//  "upstream_hc_timeoutseconds": 11
	//
	//}

	if u_in_db.Upstream_port == "" {
		u_in_db.Upstream_port = "-1"
	}

	if u_in_db.Upstream_weight == "" {
		u_in_db.Upstream_weight = "-1"
	}

	if u_in_db.Upstream_hc_intervalseconds == "" {
		u_in_db.Upstream_hc_intervalseconds = "-1"
	}

	if u_in_db.Upstream_hc_unhealthythresholdcount == "" {
		u_in_db.Upstream_hc_unhealthythresholdcount = "-1"
	}

	if u_in_db.Upstream_hc_healthythresholdcount == "" {
		u_in_db.Upstream_hc_healthythresholdcount = "-1"
	}

	if u_in_db.Upstream_hc_timeoutseconds == "" {
		u_in_db.Upstream_hc_timeoutseconds = "-1"
	}

	code3, buf3 := validate_upstream(u_in_db)

	if code3 != http.StatusOK {
		return c.JSONBlob(code3, []byte(buf3))
	}

	log.Infof(" Sending upstream values [%+v]\n", u_in_db)

	args["upstream_name"] = u_in_db.Upstream_name
	args["upstream_ip"] = u_in_db.Upstream_ip
	args["upstream_port"] = u_in_db.Upstream_port
	args["upstream_hc_path"] = u_in_db.Upstream_hc_path
	args["upstream_hc_host"] = u_in_db.Upstream_hc_host
	args["upstream_weight"] = u_in_db.Upstream_weight
	args["upstream_hc_intervalseconds"] = u_in_db.Upstream_hc_intervalseconds
	args["upstream_hc_unhealthythresholdcount"] = u_in_db.Upstream_hc_unhealthythresholdcount
	args["upstream_hc_healthythresholdcount"] = u_in_db.Upstream_hc_healthythresholdcount
	args["upstream_strategy"] = u_in_db.Upstream_strategy
	args["upstream_validation_cacertificate"] = u_in_db.Upstream_validation_cacertificate
	args["upstream_validation_subjectname"] = u_in_db.Upstream_validation_subjectname
	args["upstream_hc_timeoutseconds"] = u_in_db.Upstream_hc_timeoutseconds

	log.Infof(" Sending upstream values ARGS [%+v]\n", args)

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
		$upstream_hc_path: String!, 
		$upstream_port: Int!,
		$upstream_weight: Int
	) {
	  insert_saaras_db_upstream(
	    objects: {
	      upstream_name: $upstream_name, 
	      upstream_ip: $upstream_ip, 
	      upstream_hc_path: $upstream_hc_path, 
	      upstream_port: $upstream_port,
	      upstream_weight: $upstream_weight
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
	args["upstream_hc_path"] = u.Upstream_hc_path
	args["upstream_port"] = u.Upstream_port
	args["upstream_weight"] = u.Upstream_weight

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	log.Infof("db_insert_upstream() with [%+v]\n", buf)

	if err := saaras.RunDBQuery(url, QCreateUpstream, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return http.StatusBadRequest, buf.String()
	}

	log.Infof("db_insert_upstream() returned with status [%+s]\n", buf.String())

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

	port, err := strconv.Atoi(u.Upstream_port)

	if port < 1 || port > 65535 || err != nil {
		return http.StatusBadRequest, "Please provide a valid port value."
	}

	weight, err2 := strconv.Atoi(u.Upstream_weight)

	if weight < 0 || err2 != nil {
		return http.StatusBadRequest, "{ \"Error\" : \"Please provide a valid weight value.\" }"
	}

	// TODO: Should we make health check path mandatory? Without the path, the health checker is
	// is not programmed and it is not getting programmed on envoy through CDS/EDS
	if len(u.Upstream_hc_path) > 0 {
	} else {
		return http.StatusBadRequest, "{ \"Error\" : \"Please provide a value for upstream_hc_path\" }"
	}

	//if len(u.Upstream_hc_host) > 0 {
	//} else {
	//		  return http.StatusBadRequest, "\"Error\" : \"Please provide a value for upstream_hc_host\""
	//}

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

// @Summary Create an upstream
// @Tags upstream
// @Accept  json
// @Produce  json
// @Param Upstream body webhttp.Upstream true "Upstream to create"
// @Success 200 {} integer OK
// @Router /upstream [post]
// @Security ApiKeyAuth
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

	log.Infof("POST_Upstream() calling db_insert_upstream() with [%+v]\n", u)

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
    	upstream_hc_intervalseconds
    	upstream_hc_timeoutseconds
    	upstream_hc_unhealthythresholdcount
    	upstream_hc_healthythresholdcount
    	upstream_strategy
    	upstream_validation_cacertificate
    	upstream_validation_subjectname
		upstream_weight
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

	//{
	//    "data": {
	//        "saaras_db_upstream": [
	//            {
	//                "create_ts": "2019-09-06T16:19:51.774798+00:00",
	//                "update_ts": "2019-09-06T18:50:25.667251+00:00",
	//                "upstream_hc_healthythresholdcount": null,
	//                "upstream_hc_host": "127.0.0.1",
	//                "upstream_hc_intervalseconds": null,
	//                "upstream_hc_path": "/",
	//                "upstream_hc_timeoutseconds": null,
	//                "upstream_hc_unhealthythresholdcount": null,
	//                "upstream_id": 7,
	//                "upstream_ip": "127.0.0.1",
	//                "upstream_name": "test3",
	//                "upstream_port": 9001,
	//                "upstream_strategy": null,
	//                "upstream_validation_cacertificate": null,
	//                "upstream_validation_subjectname": null
	//            }
	//        ]
	//    }
	//}
	type SaarasDbUpstream struct {
		CreateTs                          time.Time `json:"create_ts"`
		UpdateTs                          time.Time `json:"update_ts"`
		UpstreamHcHealthythresholdcount   int       `json:"upstream_hc_healthythresholdcount"`
		UpstreamHcHost                    string    `json:"upstream_hc_host"`
		UpstreamHcIntervalseconds         int       `json:"upstream_hc_intervalseconds"`
		UpstreamHcPath                    string    `json:"upstream_hc_path"`
		UpstreamHcTimeoutseconds          int       `json:"upstream_hc_timeoutseconds"`
		UpstreamHcUnhealthythresholdcount int       `json:"upstream_hc_unhealthythresholdcount"`
		UpstreamID                        int       `json:"upstream_id"`
		UpstreamIP                        string    `json:"upstream_ip"`
		UpstreamName                      string    `json:"upstream_name"`
		UpstreamPort                      int       `json:"upstream_port"`
		UpstreamStrategy                  string    `json:"upstream_strategy"`
		UpstreamWeight                    int       `json:"upstream_weight"`
		UpstreamValidationCacertificate   string    `json:"upstream_validation_cacertificate"`
		UpstreamValidationSubjectname     string    `json:"upstream_validation_subjectname"`
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
			u.Upstream_hc_intervalseconds = strconv.FormatInt(int64(gr.Data.SaarasDbUpstream[0].UpstreamHcIntervalseconds), 10)
			u.Upstream_hc_timeoutseconds = strconv.FormatInt(int64(gr.Data.SaarasDbUpstream[0].UpstreamHcTimeoutseconds), 10)
			u.Upstream_hc_unhealthythresholdcount = strconv.FormatInt(int64(gr.Data.SaarasDbUpstream[0].UpstreamHcUnhealthythresholdcount), 10)
			u.Upstream_hc_healthythresholdcount = strconv.FormatInt(int64(gr.Data.SaarasDbUpstream[0].UpstreamHcHealthythresholdcount), 10)
			u.Upstream_strategy = gr.Data.SaarasDbUpstream[0].UpstreamStrategy
			u.Upstream_validation_cacertificate = gr.Data.SaarasDbUpstream[0].UpstreamValidationCacertificate
			u.Upstream_validation_subjectname = gr.Data.SaarasDbUpstream[0].UpstreamValidationSubjectname
			u.Upstream_weight = strconv.FormatInt(int64(gr.Data.SaarasDbUpstream[0].UpstreamWeight), 10)
		}

		log.Infof("Decoded to upstream [%v]\n", u)
	}

	return http.StatusOK, buf.String(), &u
}

// @Summary Get info of an upstream
// @Tags upstream
// @Accept  json
// @Produce  json
// @Param upstream_name path string true "Name of upstream to delete"
// @Success 200 {} integer OK
// @Router /upstream/{upstream_name} [get]
// @Security ApiKeyAuth
func GET_One_Upstream(c echo.Context) error {

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	upstream_name := c.Param("upstream_name")
	code, buf, _ := db_get_one_upstream(upstream_name, false, log)
	return c.JSONBlob(code, []byte(buf))
}

// @Summary Copy an upstream
// @Description Make a copy from upstream_name_src to upstream_name_dst
// @Tags upstream, operational-verbs
// @Accept  json
// @Produce  json
// @Param upstream_name_src path string true "Name of upstream"
// @Param upstream_name_dst path string true "Name of upstream"
// @Success 200 {} integer OK
// @Router /upstream/copy/{upstream_name_src}/{upstream_name_dst} [get]
// @Security ApiKeyAuth
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
	// TODO: Post with a query with all fields. the db_insert_upstream call right now only copies a few fields
	code2, buf2 := db_insert_upstream(u, log)
	return c.JSONBlob(code2, []byte(buf2))

}

// @Summary List all upstreams
// @Description Get a list of all upstreams
// @Tags upstream
// @Accept  json
// @Produce  json
// @Success 200 {} integer OK
// @Router /upstream [get]
// @Security ApiKeyAuth
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

// @Summary Delete an upstream
// @Tags upstream
// @Accept  json
// @Produce  json
// @Param upstream_name path string true "Name of upstream to delete"
// @Success 200 {} integer OK
// @Router /upstream/{upstream_name} [delete]
// @Security ApiKeyAuth
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

// @Summary Get routes to which this upstream is associated
// @Tags upstream, route
// @Accept  json
// @Produce  json
// @Param upstream_name path string true "Name of upstream"
// @Success 200 {} integer OK
// @Router /upstream/{upstream_name}/route [get]
// @Security ApiKeyAuth
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

	// Support for operational-verbs
	e.POST("/upstream/copy/:upstream_name_src/:upstream_name_dst", POST_Upstream_Copy)
}
