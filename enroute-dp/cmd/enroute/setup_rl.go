package main

import (
	"crypto/tls"
	"github.com/saarasio/enroute/enroute-dp/internal/grpc"
	"github.com/saarasio/enroute/enroute-dp/internal/workgroup"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
)

func SetupRateLimit(g *workgroup.Group, log logrus.FieldLogger, ctx *serveContext, c chan string) {
	log.Println("SetupRateLimit():\n")
	g.Add(func(stop <-chan struct{}) error {
		log := log.WithField("context", "grpc_ratelimit")
		addr := net.JoinHostPort(ctx.rlAddr, strconv.Itoa(ctx.rlPort))

		var l net.Listener
		var err error
		tlsconfig := ctx.tlsconfig()
		if tlsconfig != nil {
			log.Info("Setting up TLS for gRPC")
			l, err = tls.Listen("tcp", addr, tlsconfig)
			if err != nil {
				return err
			}
		} else {
			l, err = net.Listen("tcp", addr)
			if err != nil {
				return err
			}
		}

		s := grpc.NewAPIRateLimit(log, c)
		log.Println("started")
		defer log.Println("stopped")
		return s.Serve(l)
	})

}
