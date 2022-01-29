package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	containerapigrpc "github.com/nspcc-dev/neofs-api-go/v2/container/grpc"
	objectapigrpc "github.com/nspcc-dev/neofs-api-go/v2/object/grpc"
	sessionapigrpc "github.com/nspcc-dev/neofs-api-go/v2/session/grpc"
	containergrpc "github.com/nspcc-dev/neofs-node/pkg/network/transport/container/grpc"
	objectgrpc "github.com/nspcc-dev/neofs-node/pkg/network/transport/object/grpc"
	sessiongrpc "github.com/nspcc-dev/neofs-node/pkg/network/transport/session/grpc"
	"github.com/nspcc-dev/neofs-node/pkg/services/container"
	container2 "github.com/nspcc-dev/neofs-node/pkg/services/container/morph"
	"github.com/nspcc-dev/neofs-node/pkg/services/object"
	"github.com/nspcc-dev/neofs-node/pkg/services/object/acl"
	"github.com/nspcc-dev/neofs-node/pkg/services/session"
	"github.com/nspcc-dev/neofs-sdk-go/netmap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type appPreparer struct {
	cfg appConfig

	basics struct {
		key keys.PrivateKey
	}

	localNode struct {
		info netmap.NodeInfo
	}

	grpc struct {
		server *grpc.Server
	}

	network struct {
		ir struct {
			state innerRing
		}

		netMap struct {
			state netMap
		}

		containers struct {
			state containers
		}
	}

	api struct {
		object struct {
			server object.ServiceServer
		}

		session struct {
			server session.Server
		}

		container struct {
			server container.Server
		}
	}
}

type prepareAppContext struct {
	basics struct {
		keyFilepath string
	}

	network struct {
		ir struct {
			keysStr []string
		}
	}

	localNode struct {
		infoFilepath string
	}
}

func (x *appPreparer) grpcListenAddressTo(dst *string) {
	x.cfg.grpcListenAddressTo(dst)
}

func (x *appPreparer) grpcServerTo(dst *grpc.Server) {
	x.grpc.server = dst
}

func (x *appPreparer) prepare() {
	// create preparation context
	var ctxPrep prepareAppContext

	// bind config targets
	x.cfg.keyFilepathTo(&ctxPrep.basics.keyFilepath)
	x.cfg.innerRingKeysTo(&ctxPrep.network.ir.keysStr)
	x.cfg.netMapEpochTo(&x.network.netMap.state.epoch)
	x.cfg.localNodeInfoFilepathTo(&ctxPrep.localNode.infoFilepath)

	// read the config
	x.cfg.read()

	// prepare application components
	x.prepareBasics(&ctxPrep)
	x.prepareLocalNode(&ctxPrep)
	x.prepareNetwork(&ctxPrep)
	x.prepareAPI(&ctxPrep)
	x.prepareGRPC(&ctxPrep)
}

func (x *appPreparer) prepareBasics(ctx *prepareAppContext) {
	binKey, err := os.ReadFile(ctx.basics.keyFilepath)
	if err != nil {
		fmt.Errorf("read private key file: %w", err)
	}

	k, err := keys.NewPrivateKeyFromBytes(binKey)
	if err != nil {
		panic(fmt.Errorf("decode private key: %w", err))
	}

	x.basics.key = *k
}

func (x *appPreparer) prepareLocalNode(ctx *prepareAppContext) {
	jData, err := os.ReadFile(ctx.localNode.infoFilepath)
	if err != nil {
		panic(fmt.Errorf("read file with local node info: %w", err))
	}

	err = json.Unmarshal(jData, &x.localNode.info)
	if err != nil {
		panic(fmt.Errorf("decode local node info JSON: %w", err))
	}

	if x.localNode.info.State() == 0 {
		x.localNode.info.SetState(netmap.NodeStateOnline)

		log.Printf("missing local node state in JSON, set to %s\n", netmap.NodeStateOnline)
	}

	if len(x.localNode.info.PublicKey()) == 0 {
		x.localNode.info.SetPublicKey(x.basics.key.PublicKey().Bytes())

		log.Println("missing local node public key in JSON, set to local key")
	}
}

func (x *appPreparer) prepareNetwork(ctx *prepareAppContext) {
	x.prepareInnerRing(ctx)
	x.prepareNetMap(ctx)
	x.prepareContainers(ctx)
}

func (x *appPreparer) prepareInnerRing(ctx *prepareAppContext) {
	x.network.ir.state.keys = make([][]byte, len(ctx.network.ir.keysStr))

	var err error

	for i := range ctx.network.ir.keysStr {
		x.network.ir.state.keys[i], err = hex.DecodeString(ctx.network.ir.keysStr[i])
		if err != nil {
			panic(fmt.Errorf("decode IR key: %w", err))
		}
	}
}

func (x *appPreparer) prepareNetMap(_ *prepareAppContext) {
	x.network.netMap.state.nmStatic.Nodes = netmap.NodesFromInfo([]netmap.NodeInfo{x.localNode.info})
}

func (x *appPreparer) prepareContainers(_ *prepareAppContext) {
	x.network.containers.state.init()
}

func (x *appPreparer) prepareAPI(ctx *prepareAppContext) {
	x.prepareAPIObject(ctx)
	x.prepareAPISession(ctx)
	x.prepareAPIContainer(ctx)
}

func (x *appPreparer) prepareAPIObject(_ *prepareAppContext) {
	x.api.object.server = new(serviceServerObject)

	x.api.object.server = acl.New(
		acl.WithNextService(x.api.object.server),
		acl.WithSenderClassifier(
			acl.NewSenderClassifier(zap.NewNop(), &x.network.ir.state, &x.network.netMap.state),
		),
		acl.WithContainerSource(&x.network.containers.state),
		acl.WithEACLSource(&x.network.containers.state),
		acl.WithNetmapState(&x.network.netMap.state),
	)

	x.api.object.server = object.NewSignService(&x.basics.key.PrivateKey, x.api.object.server)
}

func (x *appPreparer) prepareAPIContainer(_ *prepareAppContext) {
	x.api.container.server = container.NewExecutionService(
		container2.NewExecutor(&x.network.containers.state, &x.network.containers.state),
	)

	x.api.container.server = container.NewSignService(&x.basics.key.PrivateKey, x.api.container.server)
}

func (x *appPreparer) prepareAPISession(_ *prepareAppContext) {
	x.api.session.server = new(serviceServerSession)
	x.api.session.server = session.NewSignService(&x.basics.key.PrivateKey, x.api.session.server)
}

func (x *appPreparer) prepareGRPC(_ *prepareAppContext) {
	*x.grpc.server = *grpc.NewServer()

	objectapigrpc.RegisterObjectServiceServer(x.grpc.server, objectgrpc.New(x.api.object.server))
	sessionapigrpc.RegisterSessionServiceServer(x.grpc.server, sessiongrpc.New(x.api.session.server))
	containerapigrpc.RegisterContainerServiceServer(x.grpc.server, containergrpc.New(x.api.container.server))
}
