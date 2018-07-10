package poloniex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yarencheng/crypto-trade/go/entity"
)

func TestSss(t *testing.T) {
	// arrange
	p := New()
	p.Currencies = []entity.Currency{entity.BTC, entity.ETH}
	orderBooks := make(chan entity.OrderBook)
	go func() {
		for {
			<-orderBooks
		}
	}()
	p.OrderBooks = orderBooks

	// action
	err := p.Start()
	defer p.Stop(context.Background())

	// assert
	assert.NoError(t, err)
}
