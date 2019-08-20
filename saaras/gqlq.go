package saaras

import (
	"bytes"
	"context"
	"github.com/sirupsen/logrus"
)

const SAARAS_GRAPHQL_SERVER_URL = "http://51.158.75.43/v1alpha1/graphql"

var ENROUTE_CP_SERVER_IP string

func FetchConfig(query string, buf *bytes.Buffer, args map[string]string, log logrus.FieldLogger) error {
	SAARAS_GRAPHQL_SERVER_URL2 := "http://" + ENROUTE_CP_SERVER_IP + "/v1/graphql"
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
		log.Debugf("Received buf [%s]\n", buf.String())
		return nil
	}
}
