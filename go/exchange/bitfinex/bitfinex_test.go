package bitfinex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yarencheng/crypto-trade/go/entity"
)

func Test_Start(t *testing.T) {

	// arrange
	b := New()
	b.Currencies = []entity.Currency{entity.BTC, entity.ETH}
	orderBooks := make(chan entity.OrderBook)
	go func() {
		for {
			<-orderBooks
		}
	}()
	b.OrderBooks = orderBooks

	// action
	err := b.Start()
	defer b.Stop(context.Background())

	// assert
	assert.NoError(t, err)
}