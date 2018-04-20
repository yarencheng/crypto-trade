package readonly

import (
	"testing"
	"time"

	"github.com/yarencheng/crypto-trade/data"
)

func TestNewReadOnlyStrategy(t *testing.T) {

	strategy := NewReadOnlyStrategy()

	if strategy == nil {
		t.Error()
	}

}

func TestReadOrders(t *testing.T) {

	// arrange

	strategy := NewReadOnlyStrategy()
	orders := make(chan data.Order)

	// act

	strategy.ReadOrders(orders)
	strategy.Run()

	orders <- data.Order{}

	// assert

	start := time.Now()

	for len(orders) > 0 && start.Sub(start).Seconds() < 10 {
		time.Sleep(time.Millisecond)
	}

	if len(orders) > 0 {
		t.Error("not read")
	}
}
