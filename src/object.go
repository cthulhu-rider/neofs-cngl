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

func (x *serviceServerObject) Get(req *objectV2.GetRequest, stream objectSvc.GetObjectStream) error {
	var reqLog requestProcLogger

	reqLog.name = "Object.Get"

	reqDumper.acquire(&reqLog)

	defer reqLog.free()

	printMessage(&reqLog, req)

	var partInit objectV2.GetObjectPartInit

	obj := objecttest.Object().ToV2()

	partInit.SetHeader(obj.GetHeader())
	partInit.SetObjectID(obj.GetObjectID())
	partInit.SetSignature(obj.GetSignature())

	var body objectV2.GetResponseBody

	body.SetObjectPart(&partInit)

	var resp objectV2.GetResponse

	resp.SetBody(&body)

	err := stream.Send(&resp)
	if err != nil {
		return err
	}

	var partChunk objectV2.GetObjectPartChunk

	body.SetObjectPart(&partChunk)

	txt := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore " +
		"et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip " +
		"ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu " +
		"fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt " +
		"mollit anim id est laborum.")

	for ln := len(txt); ln > 0; ln = len(txt) {
		if ln > 5 {
			ln = 5
		}

		partChunk.SetChunk(txt[:ln])

		resp.SetVerificationHeader(nil) // because we reuse same response

		err = stream.Send(&resp)
		if err != nil {
			return err
		}

		txt = txt[ln:]
	}

	return nil
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
