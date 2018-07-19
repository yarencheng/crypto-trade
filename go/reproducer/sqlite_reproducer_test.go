package reproducer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yarencheng/crypto-trade/go/entity"
)

func TestAsss(t *testing.T) {

	orders := make(chan entity.OrderBookEvent, 100)
	go func() {
		for {
			if _, ok := <-orders; !ok {
				return
			}
		}
	}()

	re := New()
	re.Path = "/home/arenx/go/src/github.com/yarencheng/crypto-trade/poloniex.order.sqlite"
	re.OrderBooks = orders

	err := re.Start()
	require.NoError(t, err)

	time.Sleep(100 * time.Second)

	t.Fail()

}
