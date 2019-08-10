package saaras

import (
	"bytes"
	"context"
	"github.com/sirupsen/logrus"
)

const SAARAS_GRAPHQL_SERVER_URL = "http://51.158.75.43/v1alpha1/graphql"

func FetchConfig(query string, buf *bytes.Buffer, args map[string]string, log logrus.FieldLogger) error {
	client := NewClient(SAARAS_GRAPHQL_SERVER_URL)
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
