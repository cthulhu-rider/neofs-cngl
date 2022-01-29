package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	v2session "github.com/nspcc-dev/neofs-api-go/v2/session"
)

type serviceServerSession struct {
}

func (x *serviceServerSession) Create(_ context.Context, req *v2session.CreateRequest) (*v2session.CreateResponse, error) {
	var reqLog requestProcLogger
	reqLog.name = "Session.Create"

	reqDumper.acquire(&reqLog)

	defer reqLog.free()

	printMessage(&reqLog, req)

	var (
		err error
		key *keys.PrivateKey
	)

	key, err = keys.NewPrivateKey()
	if err != nil {
		panic(err)
	}

	id := uuid.New()

	var binID []byte

	binID, err = id.MarshalBinary()
	if err != nil {
		panic(err)
	}

	binKey := key.PublicKey().Bytes()

	reqLog.writeString(fmt.Sprintf("generated session\n\tid: %s\n\tkey: %s",
		base64.StdEncoding.EncodeToString(binID),
		base64.StdEncoding.EncodeToString(binKey),
	))

	var bodyResp v2session.CreateResponseBody
	bodyResp.SetID(binID)
	bodyResp.SetSessionKey(binKey)

	var resp v2session.CreateResponse
	resp.SetBody(&bodyResp)

	return &resp, nil
}
