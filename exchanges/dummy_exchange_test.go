package exchanges

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetOrders_checkChannelClose(t *testing.T) {
	// arrange
	ex := NewDummyExchange(0)

	// action
	orders, e := ex.GetOrders("", "")
	require.Nil(t, e)
	ex.Finalize()
	for i := len(orders); i > 0; i-- {
		<-orders
	}

	// assert
	_, ok := <-orders
	assert.False(t, ok)
}
