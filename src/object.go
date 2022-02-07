package main

import (
	"context"
	"errors"
	"fmt"

	objectV2 "github.com/nspcc-dev/neofs-api-go/v2/object"
	objectSvc "github.com/nspcc-dev/neofs-node/pkg/services/object"
	objecttest "github.com/nspcc-dev/neofs-sdk-go/object/test"
)

type serviceServerObject struct {
}

func (x *serviceServerObject) Get(_ *objectV2.GetRequest, _ objectSvc.GetObjectStream) error {
	return errors.New("unimplemented")
}

type streamObjectPut struct {
	reqLog requestProcLogger
}

// type jsonObjectPutPayload objectV2.PutRequest
//
// func (x jsonObjectPutPayload) MarshalJSON() ([]byte, error) {
//	jTxt, err := protojson.MarshalOptions{
//		Multiline:       true,
//		EmitUnpopulated: true,
//	}.Marshal((*objectV2.PutRequest)(&x).ToGRPCMessage().(proto.Message))
//	if err != nil {
//		return nil, err
//	}
//
//	var jMap map[string]interface{}
//
//	err = json.Unmarshal(jTxt, &jMap)
//	if err != nil {
//		return nil, err
//	}
//
//	nodes := result["nodes"].([]interface{})
//	for _, node := range node {
//		m := node.(map[string]interface{})
//		if name, exists := m["name"]; exists {
//			if name == "node1" {
//				m["location"] = "new-value1"
//			} else if name == "node2" {
//				m["location"] = "new-value2"
//			}
//		}
//	}
//
//	// Convert golang object back to byte
//	byteValue, err = json.Marshal(result)
//	if err != nil {
//		return err
//	}
// }

func (x *streamObjectPut) Send(req *objectV2.PutRequest) error {
	printMessage(&x.reqLog, req)
	return nil
}

func (x *streamObjectPut) CloseAndRecv() (*objectV2.PutResponse, error) {
	defer x.reqLog.free()

	id := objecttest.ID()

	x.reqLog.writeString(fmt.Sprintf("generated object ID %s", id))

	var bodyResp objectV2.PutResponseBody
	bodyResp.SetObjectID(id.ToV2())

	var resp objectV2.PutResponse
	resp.SetBody(&bodyResp)

	return &resp, nil
}

func (x *serviceServerObject) Put(_ context.Context) (objectSvc.PutObjectStream, error) {
	var stream streamObjectPut

	stream.reqLog.name = "Object.Put"

	reqDumper.acquire(&stream.reqLog)

	return &stream, nil
}

func (x *serviceServerObject) Head(_ context.Context, _ *objectV2.HeadRequest) (*objectV2.HeadResponse, error) {
	return nil, errors.New("unimplemented")
}

func (x *serviceServerObject) Search(_ *objectV2.SearchRequest, _ objectSvc.SearchStream) error {
	return errors.New("unimplemented")
}

func (x *serviceServerObject) Delete(_ context.Context, _ *objectV2.DeleteRequest) (*objectV2.DeleteResponse, error) {
	return nil, errors.New("unimplemented")
}

func (x *serviceServerObject) GetRange(_ *objectV2.GetRangeRequest, _ objectSvc.GetObjectRangeStream) error {
	return errors.New("unimplemented")
}

func (x *serviceServerObject) GetRangeHash(_ context.Context, _ *objectV2.GetRangeHashRequest) (*objectV2.GetRangeHashResponse, error) {
	return nil, errors.New("unimplemented")
}
