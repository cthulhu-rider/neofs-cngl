package main

// application config which provides initialization parameters for the application.
type appConfig struct {
	basics struct {
		keyFilepath *string
	}

	localNode struct {
		infoFilepath *string
	}

	grpc struct {
		listenAddress *string
	}

	network struct {
		ir struct {
			keysStr *[]string
		}

		netMap struct {
			epoch *uint64
		}
	}

	storage struct {
		localObjectsFilepath *string
	}
}

func (x *appConfig) keyFilepathTo(dst *string) {
	x.basics.keyFilepath = dst
}

func (x *appConfig) localNodeInfoFilepathTo(dst *string) {
	x.localNode.infoFilepath = dst
}

func (x *appConfig) grpcListenAddressTo(dst *string) {
	x.grpc.listenAddress = dst
}

func (x *appConfig) innerRingKeysTo(dst *[]string) {
	x.network.ir.keysStr = dst
}

func (x *appConfig) netMapEpochTo(dst *uint64) {
	x.network.netMap.epoch = dst
}

func (x *appConfig) localObjectStorageFilepathTo(dst *string) {
	x.storage.localObjectsFilepath = dst
}
