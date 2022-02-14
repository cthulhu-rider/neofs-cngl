package main

import "github.com/nspcc-dev/neofs-node/cmd/neofs-node/config"

type readConfigContext struct {
	c config.Config
}

func (x *appConfig) read() {
	var fPath string

	var prmAppCfg prmAppConfig
	prmAppCfg.filepathTo(&fPath)

	prmAppCfg.read()

	if fPath == "" {
		panic("missing config filepath")
	}

	var (
		ctxRead   readConfigContext
		prmConfig config.Prm
	)

	ctxRead.c = *config.New(prmConfig,
		config.WithConfigFile(fPath),
	)

	x.readBasics(&ctxRead)
	x.readNetwork(&ctxRead)
	x.readLocalNode(&ctxRead)
	x.readGRPC(&ctxRead)
	x.readStorage(&ctxRead)
}

func (x *appConfig) readBasics(ctx *readConfigContext) {
	*x.basics.keyFilepath = config.String(&ctx.c, "basics.key.path")
}

func (x *appConfig) readLocalNode(ctx *readConfigContext) {
	*x.localNode.infoFilepath = config.String(&ctx.c, "local_node.info.path")
}

func (x *appConfig) readGRPC(ctx *readConfigContext) {
	*x.grpc.listenAddress = config.String(&ctx.c, "listen.grpc.server.endpoint")
}

func (x *appConfig) readNetwork(ctx *readConfigContext) {
	c := ctx.c.Sub("network")
	*x.network.ir.keysStr = config.StringSlice(c, "inner_ring.keys")
	*x.network.netMap.epoch = config.Uint(c, "netmap.epoch")
}

func (x *appConfig) readStorage(ctx *readConfigContext) {
	*x.storage.localObjectsFilepath = config.String(&ctx.c, "storage.path")
}
