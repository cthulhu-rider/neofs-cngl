package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/nspcc-dev/neofs-api-go/v2/rpc/message"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type requestProcLogger struct {
	name string

	unlocker interface {
		Unlock()
	}

	s strings.Builder
}

func printMessage(dst *requestProcLogger, msg interface{}) {
	var (
		jTxt []byte
		err  error
	)

	switch v := msg.(type) {
	default:
		err = fmt.Errorf("unsupported message type %T, must be json.Marshaler or message.Message", msg)
	case json.Marshaler:
		jTxt, err = v.MarshalJSON()
	case message.Message:
		jTxt, err = protojson.MarshalOptions{
			Multiline:       true,
			EmitUnpopulated: true,
		}.Marshal(v.ToGRPCMessage().(proto.Message))
	}

	if err != nil {
		panic(err)
	}

	dst.writeString(fmt.Sprintf("new request\n%s\n%T %s\n%s\n",
		"↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓",
		msg,
		string(jTxt),
		"↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑",
	))
}

func (x *requestProcLogger) writeString(s string) {
	x.s.WriteString(s + "\n")
}

func (x *requestProcLogger) free() {
	log.Println()
	log.Println(fmt.Sprintf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\nprocess %s\n\n%s", x.name, x.s.String()))
	log.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	log.Println()
	x.unlocker.Unlock()
}

type requestDumper struct {
	mtx sync.Mutex
}

func (x *requestDumper) acquire(dst *requestProcLogger) {
	x.mtx.Lock()

	dst.unlocker = &x.mtx
}

var reqDumper requestDumper
