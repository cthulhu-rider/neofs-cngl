package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"math"

	objectV2 "github.com/nspcc-dev/neofs-api-go/v2/object"
	"github.com/nspcc-dev/neofs-api-go/v2/refs"
	"github.com/nspcc-dev/neofs-node/pkg/core/netmap"
	objectcore "github.com/nspcc-dev/neofs-node/pkg/core/object"
	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/engine"
	objectSvc "github.com/nspcc-dev/neofs-node/pkg/services/object"
	"github.com/nspcc-dev/neofs-node/pkg/services/object_manager/transformer"
	"github.com/nspcc-dev/neofs-node/pkg/services/session/storage"
	"github.com/nspcc-dev/neofs-node/pkg/util"
	"github.com/nspcc-dev/neofs-sdk-go/checksum"
	cid "github.com/nspcc-dev/neofs-sdk-go/container/id"
	"github.com/nspcc-dev/neofs-sdk-go/object"
	"github.com/nspcc-dev/neofs-sdk-go/object/address"
	oid "github.com/nspcc-dev/neofs-sdk-go/object/id"
	"github.com/nspcc-dev/neofs-sdk-go/session"
	"github.com/nspcc-dev/tzhash/tz"
)

type serviceServerObject struct {
	sessionTokens *storage.TokenStore

	containers *containers

	localObjects *engine.StorageEngine

	netState netmap.State
}

// copied from neofs-node

func (x *serviceServerObject) DeleteObjects(ts *address.Address, as ...*address.Address) {
	prm := new(engine.InhumePrm)

	for _, a := range as {
		prm.WithTarget(ts, a)

		if _, err := x.localObjects.Inhume(prm); err != nil {
			log.Println("could not delete object", a, err)
		}
	}
}

func (x *serviceServerObject) onExistingContainer(id *refs.ContainerID, f func() error) error {
	_, err := x.containers.Get(cid.NewFromV2(id))
	if err != nil {
		return err
	}

	return f()
}

func (x *serviceServerObject) Get(req *objectV2.GetRequest, stream objectSvc.GetObjectStream) error {
	return x.onExistingContainer(req.GetBody().GetAddress().GetContainerID(), func() error {
		var prm engine.GetPrm
		prm.WithAddress(address.NewAddressFromV2(req.GetBody().GetAddress()))

		resLocal, err := x.localObjects.Get(&prm)
		if err != nil {
			return err
		}

		v2obj := resLocal.Object().ToV2()

		var partInit objectV2.GetObjectPartInit

		partInit.SetHeader(v2obj.GetHeader())
		partInit.SetObjectID(v2obj.GetObjectID())
		partInit.SetSignature(v2obj.GetSignature())

		var body objectV2.GetResponseBody

		body.SetObjectPart(&partInit)

		var resp objectV2.GetResponse

		resp.SetBody(&body)

		err = stream.Send(&resp)
		if err != nil {
			return err
		}

		var partChunk objectV2.GetObjectPartChunk

		body.SetObjectPart(&partChunk)

		payload := v2obj.GetPayload()

		for ln := len(payload); ln > 0; ln = len(payload) {
			if ln > 4096 {
				ln = 4086
			}

			partChunk.SetChunk(payload[:ln])

			resp.SetVerificationHeader(nil) // because we reuse same response

			err = stream.Send(&resp)
			if err != nil {
				return err
			}

			payload = payload[ln:]
		}

		return nil
	})
}

type streamObjectPut struct {
	withSession  bool
	tokenSession session.Token

	obj objectV2.Object

	svc *serviceServerObject

	id oid.ID
}

func (x *streamObjectPut) Send(req *objectV2.PutRequest) error {
	switch v := req.GetBody().GetObjectPart().(type) {
	default:
		return fmt.Errorf("unexpected object part: %T", v)
	case *objectV2.PutObjectPartInit:
		return x.svc.onExistingContainer(v.GetHeader().GetContainerID(), func() error {
			tokv2 := req.GetMetaHeader().GetSessionToken()
			if x.withSession = tokv2 != nil; x.withSession {
				x.tokenSession = *session.NewTokenFromV2(tokv2)
			}

			x.obj.SetObjectID(v.GetObjectID())
			x.obj.SetHeader(v.GetHeader())
			x.obj.SetSignature(v.GetSignature())

			return nil
		})
	case *objectV2.PutObjectPartChunk:
		x.obj.SetPayload(append(x.obj.GetPayload(), v.GetChunk()...))
	}

	return nil
}

func (x *streamObjectPut) CloseAndRecv() (*objectV2.PutResponse, error) {
	err := x.finalizeAndSave()
	if err != nil {
		return nil, err
	}

	var bodyResp objectV2.PutResponseBody
	bodyResp.SetObjectID(x.id.ToV2())

	var resp objectV2.PutResponse
	resp.SetBody(&bodyResp)

	return &resp, nil
}

// ========================= copied from neofs-node repo
type validatingTarget struct {
	nextTarget transformer.ObjectTarget

	fmt *objectcore.FormatValidator

	hash hash.Hash

	checksum []byte

	maxPayloadSz uint64 // network config

	payloadSz uint64 // payload size of the streaming object from header

	writtenPayload uint64 // number of already written payload bytes
}

// errors related to invalid payload size
var (
	errExceedingMaxSize = errors.New("payload size is greater than the limit")
	errWrongPayloadSize = errors.New("wrong payload size")
)

func (x *validatingTarget) WriteHeader(obj *objectcore.RawObject) error {
	x.payloadSz = obj.PayloadSize()
	chunkLn := uint64(len(obj.Payload()))

	// check chunk size
	if chunkLn > x.payloadSz {
		return errWrongPayloadSize
	}

	// check payload size limit
	if x.payloadSz > x.maxPayloadSz {
		return errExceedingMaxSize
	}

	cs := obj.PayloadChecksum()
	switch typ := cs.Type(); typ {
	default:
		return fmt.Errorf("(%T) unsupported payload checksum type %v", x, typ)
	case checksum.SHA256:
		x.hash = sha256.New()
	case checksum.TZ:
		x.hash = tz.New()
	}

	x.checksum = cs.Sum()

	if err := x.fmt.Validate(obj.Object()); err != nil {
		return fmt.Errorf("(%T) coult not validate object format: %w", x, err)
	}

	err := x.nextTarget.WriteHeader(obj)
	if err != nil {
		return err
	}

	// update written bytes
	//
	// Note: we MUST NOT add obj.PayloadSize() since obj
	// can carry only the chunk of the full payload
	x.writtenPayload += chunkLn

	return nil
}

func (x *validatingTarget) Write(p []byte) (n int, err error) {
	chunkLn := uint64(len(p))

	// check if new chunk will overflow payload size
	if x.writtenPayload+chunkLn > x.payloadSz {
		return 0, errWrongPayloadSize
	}

	_, err = x.hash.Write(p)
	if err != nil {
		return
	}

	n, err = x.nextTarget.Write(p)
	if err == nil {
		x.writtenPayload += uint64(n)
	}

	return
}

func (x *validatingTarget) Close() (*transformer.AccessIdentifiers, error) {
	// check payload size correctness
	if x.payloadSz != x.writtenPayload {
		return nil, errWrongPayloadSize
	}

	if !bytes.Equal(x.hash.Sum(nil), x.checksum) {
		return nil, fmt.Errorf("(%T) incorrect payload checksum", x)
	}

	return x.nextTarget.Close()
}

type localTarget struct {
	storage *engine.StorageEngine

	obj *objectcore.RawObject

	payload []byte
}

func (x *localTarget) WriteHeader(obj *objectcore.RawObject) error {
	x.obj = obj
	return nil
}

func (x *localTarget) Write(p []byte) (n int, err error) {
	x.payload = append(x.payload, p...)
	return len(p), nil
}

func (x *localTarget) Close() (*transformer.AccessIdentifiers, error) {
	x.obj.SetPayload(x.payload)

	if err := engine.Put(x.storage, x.obj.Object()); err != nil {
		return nil, fmt.Errorf("(%T) could not put object to local storage: %w", x, err)
	}

	return new(transformer.AccessIdentifiers).
		WithSelfID(x.obj.ID()), nil
}

// ===================================================

func (x *streamObjectPut) finalizeAndSave() error {
	obj := objectcore.NewRawFromV2(&x.obj)

	tgtLocal := &localTarget{
		storage: x.svc.localObjects,
	}

	var tgt transformer.ObjectTarget

	if obj.Signature() == nil {
		idOwner := obj.OwnerID()
		if idOwner == nil {
			return errors.New("missing owner in raw object")
		}

		tokenPriv := x.svc.sessionTokens.Get(idOwner, x.tokenSession.ID())
		if tokenPriv == nil {
			return errors.New("private session not found")
		} else if tokenPriv.ExpiredAt() <= x.svc.netState.CurrentEpoch() {
			return errors.New("expired session")
		}

		tgt = transformer.NewPayloadSizeLimiter(math.MaxUint64, func() transformer.ObjectTarget {
			return transformer.NewFormatTarget(&transformer.FormatterParams{
				Key:          tokenPriv.SessionKey(),
				NextTarget:   tgtLocal,
				SessionToken: &x.tokenSession,
				NetworkState: x.svc.netState,
			})
		})

		err := tgt.WriteHeader(obj)
		if err != nil {
			return fmt.Errorf("write header: %w", err)
		}

		_, err = tgt.Write(obj.Payload())
		if err != nil {
			return fmt.Errorf("write chunk: %w", err)
		}
	} else {
		tgt = &validatingTarget{
			nextTarget: tgtLocal,
			fmt: objectcore.NewFormatValidator(
				objectcore.WithNetState(x.svc.netState),
				objectcore.WithDeleteHandler(x.svc),
			),
		}

		err := tgt.WriteHeader(obj)
		if err != nil {
			return fmt.Errorf("write header: %w", err)
		}

		_, err = tgt.Write(obj.Payload())
		if err != nil {
			return fmt.Errorf("write chunk: %w", err)
		}
	}

	ids, err := tgt.Close()
	if err != nil {
		return fmt.Errorf("finalize saving: %w", err)
	}

	x.id = *ids.SelfID()

	return nil
}

func (x *serviceServerObject) Put(_ context.Context) (objectSvc.PutObjectStream, error) {
	return &streamObjectPut{
		svc: x,
	}, nil
}

func (x *serviceServerObject) Head(_ context.Context, req *objectV2.HeadRequest) (resp *objectV2.HeadResponse, err error) {
	err = x.onExistingContainer(req.GetBody().GetAddress().GetContainerID(), func() error {
		bodyReq := req.GetBody()

		addr := bodyReq.GetAddress()

		var prm engine.HeadPrm
		prm.WithAddress(address.NewAddressFromV2(addr))
		prm.WithRaw(bodyReq.GetRaw())

		res, err := x.localObjects.Head(&prm)
		if err != nil {
			return fmt.Errorf("local head: %w", err)
		}

		v2obj := res.Header().ToV2()

		var part objectV2.HeaderWithSignature

		part.SetHeader(v2obj.GetHeader())
		part.SetSignature(v2obj.GetSignature())

		var body objectV2.HeadResponseBody

		body.SetHeaderPart(&part)

		resp = new(objectV2.HeadResponse)

		resp.SetBody(&body)

		return nil
	})

	return
}

func (x *serviceServerObject) Search(req *objectV2.SearchRequest, stream objectSvc.SearchStream) error {
	return x.onExistingContainer(req.GetBody().GetContainerID(), func() error {
		var prm engine.SelectPrm
		prm.WithContainerID(cid.NewFromV2(req.GetBody().GetContainerID()))
		prm.WithFilters(object.NewSearchFiltersFromV2(req.GetBody().GetFilters()))

		res, err := x.localObjects.Select(&prm)
		if err != nil {
			return fmt.Errorf("local select: %w", err)
		}

		list := res.AddressList()

		listv2 := make([]*refs.ObjectID, 0, len(list))

		for i := range list {
			listv2 = append(listv2, list[i].ObjectID().ToV2())
		}

		var bodyResp objectV2.SearchResponseBody

		var resp objectV2.SearchResponse

		resp.SetBody(&bodyResp)

		if len(listv2) > 0 {
			bodyResp.SetIDList(listv2)
			return stream.Send(&resp)
		}

		return nil
	})
}

func (x *serviceServerObject) Delete(_ context.Context, _ *objectV2.DeleteRequest) (*objectV2.DeleteResponse, error) {
	return nil, errors.New("unimplemented")
}

func (x *serviceServerObject) GetRange(req *objectV2.GetRangeRequest, stream objectSvc.GetObjectRangeStream) error {
	return x.onExistingContainer(req.GetBody().GetAddress().GetContainerID(), func() error {
		bodyReq := req.GetBody()

		var prm engine.RngPrm
		prm.WithAddress(address.NewAddressFromV2(bodyReq.GetAddress()))
		prm.WithPayloadRange(object.NewRangeFromV2(bodyReq.GetRange()))

		res, err := x.localObjects.GetRange(&prm)
		if err != nil {
			return err
		}

		payload := res.Object().Payload()

		var partChunk objectV2.GetRangePartChunk

		var body objectV2.GetRangeResponseBody

		body.SetRangePart(&partChunk)

		var resp objectV2.GetRangeResponse

		resp.SetBody(&body)

		for ln := len(payload); ln > 0; ln = len(payload) {
			if ln > 4096 {
				ln = 4086
			}

			partChunk.SetChunk(payload[:ln])

			resp.SetVerificationHeader(nil) // because we reuse same response

			err = stream.Send(&resp)
			if err != nil {
				return err
			}

			payload = payload[ln:]
		}

		return nil
	})
}

func (x *serviceServerObject) GetRangeHash(_ context.Context, req *objectV2.GetRangeHashRequest) (resp *objectV2.GetRangeHashResponse, err error) {
	err = x.onExistingContainer(req.GetBody().GetAddress().GetContainerID(), func() error {
		bodyReq := req.GetBody()

		var prm engine.GetPrm
		prm.WithAddress(address.NewAddressFromV2(bodyReq.GetAddress()))

		res, err := x.localObjects.Get(&prm)
		if err != nil {
			return fmt.Errorf("local get: %w", err)
		}

		typ := bodyReq.GetType()
		salt := bodyReq.GetSalt()
		payload := res.Object().Payload()
		var hs [][]byte
		var w io.Writer
		var h hash.Hash
		var off, ln uint64

		switch typ {
		default:
			return fmt.Errorf("unsupported checksum type: %v", typ)
		case refs.TillichZemor:
			h = tz.New()
		case refs.SHA256:
			h = sha256.New()
		}

		for _, rng := range bodyReq.GetRanges() {
			off, ln = rng.GetOffset(), rng.GetLength()
			if off+ln > uint64(len(payload)) {
				return objectcore.ErrRangeOutOfBounds
			}

			w = util.NewSaltingWriter(h, salt)

			_, _ = w.Write(payload[off : off+ln])

			hs = append(hs, h.Sum(nil))

			h.Reset()
		}

		var body objectV2.GetRangeHashResponseBody

		body.SetHashList(hs)

		resp = new(objectV2.GetRangeHashResponse)

		resp.SetBody(&body)

		return nil
	})

	return
}
