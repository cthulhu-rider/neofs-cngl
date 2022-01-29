package main

import "github.com/nspcc-dev/neofs-sdk-go/netmap"

type netMap struct {
	epoch uint64

	nmStatic netmap.Netmap
}

func (x *netMap) GetNetMap(diff uint64) (*netmap.Netmap, error) {
	return &x.nmStatic, nil
}

func (x *netMap) GetNetMapByEpoch(epoch uint64) (*netmap.Netmap, error) {
	return &x.nmStatic, nil
}

func (x *netMap) Epoch() (uint64, error) {
	return x.epoch, nil
}

func (x *netMap) CurrentEpoch() uint64 {
	return x.epoch
}
