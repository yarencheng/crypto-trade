package poloniex

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yarencheng/crypto-trade/go/entity"
)

func TestSss(t *testing.T) {
	// arrange
	p := New()
	p.Currencies = []entity.Currency{entity.BTC, entity.ETH}
	orderBooks := make(chan entity.OrderBookEvent)
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

	time.Sleep(10 * time.Second)
}
