package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/engine"
	"google.golang.org/grpc"
)

type app struct {
	grpc struct {
		server grpc.Server
	}

	storage struct {
		objects engine.StorageEngine
	}
}

func (x *app) start() {
	defer x.release()

	chAwait := make(chan struct{})

	go func() {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer cancel()

		<-ctx.Done()

		log.Println("interrupt application on OS signal")

		close(chAwait)
	}()

	log.Println("starting application...")

	var starter appStarter
	starter.grpcServerTo(&x.grpc.server)
	starter.localObjectStorageTo(&x.storage.objects)

	starter.start()

	log.Println("application started, waiting for OS signal...")

	<-chAwait
}

func (x *app) release() {
	_ = x.storage.objects.Close()
	x.grpc.server.GracefulStop()
}
