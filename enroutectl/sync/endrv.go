// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/saarasio/enroute/enroutectl/config"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"sort"
)

type Method int

const (
	GET Method = iota
	POST
	DELETE
	PATCH
)

type HttpMethod struct {
	method Method
	arg    interface{}
	mesg   string
}

type Url string

type PostProxyArg struct {
	Name string `json:"Name"`
}

type PostServiceArg struct {
	Service_Name string `json: Service_Name`
	Fqdn         string `json: Fqdn`
}

type PostRouteArg struct {
	Route_Name   string `json: Route_Name`
	Route_Prefix string `json: Route_Prefix`
	Route_Config string `json: Route_Config`
}

type PostUpstreamArg struct {
	Upstream_name    string `json: Upstream_name`
	Upstream_ip      string `json: Upstream_ip`
	Upstream_port    string `json: Upstream_port`
	Upstream_hc_path string `json: Upstream_hc_path`
	Upstream_weight  string `json: Upstream_weight`
}

type PostFilterArg struct {
	Filter_name   string `json: Filter_name `
	Filter_type   string `json: Filter_type`
	Filter_config string `json: Filter_config`
}

type PostGlobalConfigArg struct {
	Globalconfig_name string `json: Globalconfig_name `
	Globalconfig_type string `json: Globalconfig_type`
	Config            string `json: Config`
}

const Base_url = `http://localhost:1323`

//// EnDrv ////
type EnDrv struct {
	Dbg      bool
	Base_url string
	Offline  bool
	Id       string
}

type EnStatus struct {
	ret_code int
	status   []string
	pua      *PostUpstreamArg
	psa      *PostServiceArg
	pra      *PostRouteArg
}

func (ed *EnDrv) runHttp(cmds *map[int]map[HttpMethod]Url) *EnStatus {
	status := EnStatus{
		ret_code: 200,
	}

	dohttp(cmds, ed.Dbg)

	return &status
}

func (ed *EnDrv) EnrouteAddUpstream(post_upstream_arg *PostUpstreamArg) *EnStatus {

	var urls = map[string]string{
		// Create Upstream
		"CREATE_U": ed.Base_url + "/upstream",
	}

	steps := map[int]map[HttpMethod]Url{
		125: map[HttpMethod]Url{HttpMethod{POST, &post_upstream_arg, "-- POST U --"}: Url(urls["CREATE_U"])},
	}

	if ed.Offline {
		return &EnStatus{
			pua: post_upstream_arg,
		}
	} else {
		return ed.runHttp(&steps)
	}
}

func (ed *EnDrv) EnrouteDeleteUpstream(upstream_name string) *EnStatus {

	var urls = map[string]string{
		// Delete upstream
		"DEL_SVC_U": ed.Base_url + "/upstream/" + upstream_name,
	}

	steps := map[int]map[HttpMethod]Url{
		50: map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL U --"}: Url(urls["DEL_SVC_U"])},
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteAssociateUpstreamToServiceRoute(svc_name, route_name, upstream_name string) *EnStatus {
	var urls = map[string]string{
		"SVC_U": ed.Base_url +
			"/service/" + svc_name +
			"/route/" + route_name +
			"/upstream/" + upstream_name,
	}
	steps := map[int]map[HttpMethod]Url{
		130: map[HttpMethod]Url{HttpMethod{POST, nil, "-- POST SVC/R/U --"}: Url(urls["SVC_U"])},
	}

	if ed.Offline {
		return &EnStatus{}
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) en_create_u_for_s_r(svc_name, route_name string, post_upstream_arg *PostUpstreamArg) *EnStatus {

	var urls = map[string]string{
		// Create Upstream
		"CREATE_U": ed.Base_url + "/upstream",
		"SVC_U": ed.Base_url +
			"/service/" + svc_name +
			"/route/" + route_name +
			"/upstream/" + post_upstream_arg.Upstream_name,
	}

	steps := map[int]map[HttpMethod]Url{
		125: map[HttpMethod]Url{HttpMethod{POST, &post_upstream_arg, "-- POST U --"}: Url(urls["CREATE_U"])},
		130: map[HttpMethod]Url{HttpMethod{POST, nil, "-- POST SVC/R/U --"}: Url(urls["SVC_U"])},
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteCreateRoute(service_name string, post_route_arg *PostRouteArg) *EnStatus {

	var urls = map[string]string{
		// Create Service
		"CREATE_RT": ed.Base_url +
			"/service/" + service_name +
			"/route",
	}

	steps := map[int]map[HttpMethod]Url{
		100: map[HttpMethod]Url{HttpMethod{POST, &post_route_arg, "-- POST RT --"}: Url(urls["CREATE_RT"])},
	}

	if ed.Offline {
		return &EnStatus{
			pra: post_route_arg,
		}
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteDeleteRoute(service_name, route_name string) *EnStatus {

	var urls = map[string]string{
		// Delete route
		"DEL_RT": ed.Base_url +
			"/service/" + service_name +
			"/route/" + route_name,
	}

	steps := map[int]map[HttpMethod]Url{
		75: map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL RT --"}: Url(urls["DEL_RT"])},
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteDeleteService(service_name string) *EnStatus {

	var urls = map[string]string{
		// Delete service
		"DEL_SVC": Base_url +
			"/service/" + service_name,
	}

	steps := map[int]map[HttpMethod]Url{
		100: map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL SVC --"}: Url(urls["DEL_SVC"])},
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteCreateService(post_svc_arg *PostServiceArg) *EnStatus {

	var urls = map[string]string{
		// Create Service
		"CREATE_SVC": ed.Base_url + "/service",
	}

	steps := map[int]map[HttpMethod]Url{
		75: map[HttpMethod]Url{HttpMethod{POST, &post_svc_arg, "-- POST SVC --"}: Url(urls["CREATE_SVC"])},
	}

	if ed.Offline {
		return &EnStatus{
			psa: post_svc_arg,
		}
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteCreateFilter() *EnStatus {
	status := EnStatus{
		ret_code: 200,
	}

	return &status
}

func (ed *EnDrv) EnrouteCreateGlobalconfig() *EnStatus {
	status := EnStatus{
		ret_code: 200,
	}

	return &status
}

func (ed *EnDrv) EnrouteGetStatus() *EnStatus {

	var urls = map[string]string{
		// Create Proxy
		"STATUS_G": "/status/" + ed.Id,
	}

	steps := map[int]map[HttpMethod]Url{
		75: map[HttpMethod]Url{HttpMethod{GET, nil, "-- POST P --"}: Url(ed.Base_url + urls["STATUS_G"])},
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteCreateProxy(post_p_arg *PostProxyArg) *EnStatus {

	var urls = map[string]string{
		// Create Proxy
		"CREATE_P": "/proxy",
	}

	steps := map[int]map[HttpMethod]Url{
		75: map[HttpMethod]Url{HttpMethod{POST, &post_p_arg, "-- POST P --"}: Url(ed.Base_url + urls["CREATE_P"])},
	}

	return ed.runHttp(&steps)
}

func (ed *EnDrv) EnrouteAssociateProxyService(svc string) *EnStatus {

	var urls = map[string]string{
		// Create Proxy
		"ASSO_P": "/proxy/gw/service/" + svc,
	}

	steps := map[int]map[HttpMethod]Url{
		75: map[HttpMethod]Url{HttpMethod{POST, nil, "-- ASSOCIATE P S --"}: Url(ed.Base_url + urls["ASSO_P"])},
	}

	return ed.runHttp(&steps)
}

func doreq2(req *http.Request, err error, url string, dbg bool) []byte {
	var b []byte
	if err == nil {
		client := &http.Client{}
		if dbg {
			debug(httputil.DumpRequestOut(req, true))
		}
		res, err := client.Do(req)
		if res == nil {
			// TODO: Better error reporting
			return b
		}
		body, err := ioutil.ReadAll(res.Body)
		// fmt.Printf("%s\n", pp(body))
		res.Body.Close()
		err_out(res, err)
		return body
	} else {
		fmt.Printf("Request run error while running url - [%s]\n", url)
	}
	return b
}

func pp(in []byte) string {
	var pj bytes.Buffer
	json.Indent(&pj, in, "", "    ")
	return string(pj.Bytes())
}

func err_out(res *http.Response, err error) {
	if err != nil {
		fmt.Print("Error\n")
	} else {
		b, _ := ioutil.ReadAll(res.Body)
		fmt.Println(pp(b))
	}
}

func err_check(err error, mesg string) {
	if err != nil {
		fmt.Println("Error " + mesg + " \n")
	}
}

func debug(data []byte, err error) {
	if err == nil {
		fmt.Printf("%s\n\n", data)
	} else {
		log.Fatalf("%s\n\n", err)
	}
}

func doreq(req *http.Request, err error, rstr, url string, dbg bool) {
	if err == nil {
		client := &http.Client{}
		if dbg {
			debug(httputil.DumpRequestOut(req, true))
		}
		res, err := client.Do(req)
		if res == nil {
			// TODO: Better error reporting
			// request failed
			fmt.Printf("Request run error while running [%s] url - [%s]\n", rstr, url)
			return
		}
		body, err := ioutil.ReadAll(res.Body)
		fmt.Printf("%s\n", pp(body))
		res.Body.Close()
		err_out(res, err)
	} else {
		fmt.Printf("Request run error while running [%s] url - [%s]\n", rstr, url)
	}

}

// run http commands in sequence
func dohttp(cmds *map[int]map[HttpMethod]Url, dbg bool) {
	var keys []int
	for k := range *cmds {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		c := (*cmds)[k]
		for httpmethod, url := range c {
			fmt.Println(httpmethod.mesg)
			switch httpmethod.method {
			case GET:
				req, err := http.NewRequest("GET", string(url), nil)
				doreq(req, err, "GET", string(url), dbg)

			case POST:
				var req *http.Request
				var err error
				if httpmethod.arg != nil {
					post_arg := httpmethod.arg
					post_arg_json, _ := json.Marshal(post_arg)
					req, err = http.NewRequest("POST", string(url), bytes.NewBuffer(post_arg_json))
					if err == nil {
						req.Header.Add("Content-Type", "application/json")
					} else {
						fmt.Printf("Request creation error while running POST url - [%s]\n", url)
					}

				} else {
					req, err = http.NewRequest("POST", string(url), nil)
				}

				doreq(req, err, "POST", string(url), dbg)

			case DELETE:
				req, err := http.NewRequest("DELETE", string(url), nil)
				doreq(req, err, "DELETE", string(url), dbg)

			case PATCH:
			default:
				// not supported
			}

		}
	}
}

func getsyncproxy() *config.EnrouteConfig {
	url := "http://localhost:1323/proxy/dump/gw"
	dbg := false
	req, err := http.NewRequest("GET", string(url), nil)
	resbytes := doreq2(req, err, string(url), dbg)

	buf := bytes.NewBuffer(resbytes)

	var sp config.EnrouteConfig
	if err := json.NewDecoder(buf).Decode(&sp); err != nil {
		fmt.Printf("Error when decoding json [%v]\n", err)
	}

	return &sp

	//fmt.Printf("%# v", pretty.Formatter(sp))
	// fmt.Printf("%# v", sp)
}
