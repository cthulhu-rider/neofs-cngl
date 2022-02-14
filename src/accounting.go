package main

import (
	"context"
	"math/rand"

	"github.com/nspcc-dev/neofs-api-go/v2/accounting"
)

type serviceServerAccounting struct {
	bal accounting.Decimal
}

func (x *serviceServerAccounting) Balance(context.Context, *accounting.BalanceRequest) (*accounting.BalanceResponse, error) {
	x.bal.SetValue(rand.Int63())
	x.bal.SetPrecision(rand.Uint32())

	var body accounting.BalanceResponseBody

	body.SetBalance(&x.bal)

	var resp accounting.BalanceResponse

	resp.SetBody(&body)

	return &resp, nil
}
