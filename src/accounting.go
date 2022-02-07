package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/nspcc-dev/neofs-api-go/v2/accounting"
)

type serviceServerAccounting struct {
	bal accounting.Decimal
}

func (x *serviceServerAccounting) Balance(_ context.Context, req *accounting.BalanceRequest) (*accounting.BalanceResponse, error) {
	var reqLog requestProcLogger
	reqLog.name = "Accounting.Balance"

	reqDumper.acquire(&reqLog)

	defer reqLog.free()

	printMessage(&reqLog, req)

	x.bal.SetValue(rand.Int63())
	x.bal.SetPrecision(rand.Uint32())

	reqLog.writeString(fmt.Sprintf("prepare random balance %d:%d", x.bal.GetValue(), x.bal.GetPrecision()))

	var body accounting.BalanceResponseBody

	body.SetBalance(&x.bal)

	var resp accounting.BalanceResponse

	resp.SetBody(&body)

	return &resp, nil
}
