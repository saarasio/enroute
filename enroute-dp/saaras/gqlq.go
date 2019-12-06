// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2019 Saaras Inc.


package saaras

import (
	"bytes"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

var ENROUTE_CP_SERVER_IP string
var ENROUTE_CP_SERVER_PORT string
var ENROUTE_CP_PROTO string

// Used by enroute-dp
func FetchConfig(query string, buf *bytes.Buffer, args map[string]string, log logrus.FieldLogger) error {

	var SAARAS_GRAPHQL_SERVER_URL2 string

	if ENROUTE_CP_PROTO == "HTTP" || ENROUTE_CP_PROTO == "http" {
		SAARAS_GRAPHQL_SERVER_URL2 = "http://" + ENROUTE_CP_SERVER_IP + ":" + ENROUTE_CP_SERVER_PORT + "/v1/graphql"
	} else if ENROUTE_CP_PROTO == "HTTPS" || ENROUTE_CP_PROTO == "https" {
		SAARAS_GRAPHQL_SERVER_URL2 = "https://" + ENROUTE_CP_SERVER_IP + ":" + ENROUTE_CP_SERVER_PORT + "/v1/graphql"
	} else {
		fmt.Printf("Please provide a valid value for enroute-cp-proto. Allowed values are HTTP or HTTPS\n")
		os.Exit(1)
	}

	client := NewClient(SAARAS_GRAPHQL_SERVER_URL2)
	client.Log = func(s string) { log.Debugf("%s", s) }
	req := NewRequest(query)

	for k, v := range args {
		req.Var(k, v)
	}

	ctx := context.Background()

	if err := client.Run(ctx, req, buf); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return err
	} else {
		// TODO: Fix logging
		//log.Debugf("Received buf [%s]\n", buf.String())
		return nil
	}
}

// Used by webapp
func RunDBQuery(url string, query string, buf *bytes.Buffer, args map[string]string, log logrus.FieldLogger) error {
	client := NewClient(url)
	client.Log = func(s string) { log.Debugf("%s", s) }
	req := NewRequest(query)

	for k, v := range args {
		req.Var(k, v)
	}

	ctx := context.Background()

	if err := client.Run(ctx, req, buf); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		return err
	} else {
		log.Debugf("Received buf [%s]\n", buf.String())
		return nil
	}
}
