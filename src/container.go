package main

import (
	"sync"

	containercore "github.com/nspcc-dev/neofs-node/pkg/core/container"
	"github.com/nspcc-dev/neofs-sdk-go/container"
	cid "github.com/nspcc-dev/neofs-sdk-go/container/id"
	"github.com/nspcc-dev/neofs-sdk-go/eacl"
	"github.com/nspcc-dev/neofs-sdk-go/owner"
)

type vContainer struct {
	id *cid.ID

	cnr *container.Container
}

type containers struct {
	mtxContainers sync.RWMutex
	mContainers   map[string]vContainer

	mtxEACL sync.RWMutex
	mEACL   map[string]*eacl.Table
}

func (x *containers) init() {
	x.mContainers = make(map[string]vContainer)
	x.mEACL = make(map[string]*eacl.Table)
}

func (x *containers) Delete(witness containercore.RemovalWitness) error {
	strID := witness.ContainerID().String()

	x.mtxContainers.Lock()
	delete(x.mContainers, strID)
	x.mtxContainers.Unlock()

	x.mtxEACL.Lock()
	delete(x.mEACL, strID)
	x.mtxEACL.Unlock()

	return nil
}

func (x *containers) PutEACL(table *eacl.Table) error {
	x.mtxEACL.Lock()
	x.mEACL[table.CID().String()] = table
	x.mtxEACL.Unlock()

	return nil
}

func (x *containers) Put(cnr *container.Container) (*cid.ID, error) {
	id := container.CalculateID(cnr)

	x.mtxContainers.Lock()

	x.mContainers[id.String()] = vContainer{
		id:  id,
		cnr: cnr,
	}

	x.mtxContainers.Unlock()

	return id, nil
}

func (x *containers) List(id *owner.ID) ([]*cid.ID, error) {
	x.mtxContainers.RLock()

	var res []*cid.ID

	for _, v := range x.mContainers {
		if id.Equal(v.cnr.OwnerID()) {
			res = append(res, v.id)
		}
	}

	x.mtxContainers.RUnlock()

	return res, nil
}

func (x *containers) Get(id *cid.ID) (*container.Container, error) {
	x.mtxContainers.RLock()
	defer x.mtxContainers.RUnlock()

	v, ok := x.mContainers[id.String()]
	if !ok {
		return nil, containercore.ErrNotFound
	}

	return v.cnr, nil
}

func (x *containers) GetEACL(id *cid.ID) (*eacl.Table, error) {
	x.mtxEACL.RLock()
	defer x.mtxEACL.RUnlock()

	table, ok := x.mEACL[id.String()]
	if !ok {
		return nil, containercore.ErrEACLNotFound
	}

	return table, nil
}
