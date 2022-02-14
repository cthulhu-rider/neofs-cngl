package main

import (
	"log"
	"net"

	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/engine"
	"google.golang.org/grpc"
)

type appStarter struct {
	grpc struct {
		listenAddress string

		server *grpc.Server
	}

	storage struct {
		localObjects *engine.StorageEngine
	}
}

func (x *appStarter) grpcServerTo(dst *grpc.Server) {
	x.grpc.server = dst
}

func (x *appStarter) localObjectStorageTo(dst *engine.StorageEngine) {
	x.storage.localObjects = dst
}

func (x *appStarter) start() {
	log.Println("preparing resources...")

	var prep appPreparer
	prep.grpcServerTo(x.grpc.server)
	prep.grpcListenAddressTo(&x.grpc.listenAddress)
	prep.localObjectStorageTo(x.storage.localObjects)

	prep.prepare()

	if x.grpc.listenAddress == "" {
		panic("missing gRPC listen endpoint")
	}

	log.Println("all components are ready")

	x.startLocalObjectStorage()
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

func (x *appStarter) startLocalObjectStorage() {
	err := x.storage.localObjects.Open()
	if err != nil {
		log.Fatalf("open object storage: %v", err)
	}

	err = x.storage.localObjects.Init()
	if err != nil {
		log.Fatalf("init object storage: %v", err)
	}

}
