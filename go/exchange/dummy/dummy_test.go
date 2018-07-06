package dummy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yarencheng/crypto-trade/go/data"
	"github.com/yarencheng/gospring/applicationcontext"
	"github.com/yarencheng/gospring/v1"
)

func TestSss(t *testing.T) {
	ctx := applicationcontext.Default()

	err := ctx.AddConfigs(&v1.Bean{
		ID:        "dummy_exchange",
		Type:      v1.T(Dummy{}),
		FactoryFn: New,
		Properties: []v1.Property{
			{
				Name:   "DelayMs",
				Config: v1.V(int64(1000)),
			},
			{
				Name:   "LiveOrders",
				Config: "orders",
			},
		},
	}, /*&v1.Bean{
			ID:        "stupid_strategy",
			Type:      v1.T(strategies.StupidStrategy{}),
			FactoryFn: strategies.New,
			Properties: []v1.Property{
				{
					Name:   "LiveOrders",
					Config: "ordersBroadcast",
				},
			},
		}, */&v1.Channel{
			ID:   "orders",
			Type: v1.T(data.Order{}),
			Size: 10,
		}, &v1.Broadcast{
			ID:       "ordersBroadcast",
			SourceID: "orders",
			Size:     10,
		})

	require.NoError(t, err)

	_, err = ctx.GetByID("dummy_exchange")
	require.NoError(t, err)

	b, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = ctx.Stop(b)
	require.NoError(t, err)
}
