package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
)

type appStarter struct {
	grpc struct {
		listenAddress string

		server *grpc.Server
	}
}

func (x *appStarter) grpcServerTo(dst *grpc.Server) {
	x.grpc.server = dst
}

func (x *appStarter) start() {
	log.Println("preparing resources...")

	var prep appPreparer
	prep.grpcServerTo(x.grpc.server)
	prep.grpcListenAddressTo(&x.grpc.listenAddress)

	prep.prepare()

	if x.grpc.listenAddress == "" {
		panic("missing gRPC listen endpoint")
	}

	log.Println("all components are ready")

	x.startGRPC()
}

func (x *appStarter) startGRPC() {
	lis, err := net.Listen("tcp", x.grpc.listenAddress)
	if err != nil {
		panic(err)
	}

	log.Println("listen tcp on", x.grpc.listenAddress)

	// FIXME: must be closed on accidental abort

	go func() {
		log.Println("serve gRPC on", x.grpc.listenAddress)
		if err := x.grpc.server.Serve(lis); err != nil {
			// TODO: log
		}
	}()
}
