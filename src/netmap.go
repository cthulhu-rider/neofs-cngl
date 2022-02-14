package main

import (
	"context"
	"encoding/binary"
	"time"

	netmapv2 "github.com/nspcc-dev/neofs-api-go/v2/netmap"
	"github.com/nspcc-dev/neofs-sdk-go/netmap"
)

type netMap struct {
	epoch uint64

	nmStatic netmap.Netmap
}

func (x *netMap) LocalNodeInfo(_ context.Context, _ *netmapv2.LocalNodeInfoRequest) (*netmapv2.LocalNodeInfoResponse, error) {
	return new(netmapv2.LocalNodeInfoResponse), nil
}

func (x *netMap) NetworkInfo(_ context.Context, _ *netmapv2.NetworkInfoRequest) (*netmapv2.NetworkInfoResponse, error) {
	bufEpochDur := make([]byte, 8)

	binary.LittleEndian.PutUint64(bufEpochDur, 20)

	var netPrmEpochDur netmapv2.NetworkParameter

	netPrmEpochDur.SetKey([]byte("EpochDuration"))
	netPrmEpochDur.SetValue(bufEpochDur)

	var netCfg netmapv2.NetworkConfig

	netCfg.SetParameters(&netPrmEpochDur)

	var netInfo netmapv2.NetworkInfo

	netInfo.SetMagicNumber(1337)
	netInfo.SetCurrentEpoch(x.epoch)
	netInfo.SetMsPerBlock(int64(15 * time.Second / time.Millisecond))
	netInfo.SetNetworkConfig(&netCfg)

	var body netmapv2.NetworkInfoResponseBody

	body.SetNetworkInfo(&netInfo)

	var resp netmapv2.NetworkInfoResponse

	resp.SetBody(&body)

	return &resp, nil
}

func (x *netMap) GetNetMap(_ uint64) (*netmap.Netmap, error) {
	return &x.nmStatic, nil
}

func (x *netMap) GetNetMapByEpoch(_ uint64) (*netmap.Netmap, error) {
	return &x.nmStatic, nil
}

func (x *netMap) Epoch() (uint64, error) {
	return x.epoch, nil
}

func (x *netMap) CurrentEpoch() uint64 {
	return x.epoch
}
