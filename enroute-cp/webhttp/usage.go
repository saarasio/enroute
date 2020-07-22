package webhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// TODO: Seperate envoy, enroute stats
type RaconPayload struct {
	EnrouteCtlUUID string `json:"enroutectluuid,omitempty"`
	Uptime         string `json:"uptime,omitempty"`
}

var startTime time.Time

func uptime() string {
	return time.Since(startTime).String()
}

func init() {
	startTime = time.Now()
}

func sendOnce() {

	var req *http.Request
	var err error
	client := &http.Client{}

	post_arg := RaconPayload{
		EnrouteCtlUUID: ID,
		Uptime:         uptime(),
	}

	post_arg_json, _ := json.Marshal(post_arg)

	url := "https://racon.universalapigateway.com"
	req, err = http.NewRequest("POST", string(url), bytes.NewBuffer(post_arg_json))
	if err == nil {
		req.Header.Add("Content-Type", "application/json")
	} else {
		fmt.Printf("Request creation error while running POST url - [%s]\n", url)
	}

	res, err := client.Do(req)
	if res == nil {
		// TODO: Better error reporting
		// request failed
		fmt.Printf("Request run error while running url - [%s]\n", url)
		return
	}
	_, err2 := ioutil.ReadAll(res.Body)

	fmt.Printf("Done: Post to url - [%s]\n", url)
	if err2 != nil {
		fmt.Printf("Error when reading response body [%s] \n", err2)
	}
	res.Body.Close()
}

func Reporter() {
	USAGE = os.Getenv("SEND_ANON_STAT")

	for {
		if USAGE != "no" {
			sendOnce()
		} else {
			fmt.Printf("Not sending anon stat\n")
		}
		time.Sleep(60 * 60 * time.Minute)
	}
}
