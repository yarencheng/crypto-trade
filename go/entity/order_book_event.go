package entity

type OrderBookEventType string

const (
	ExchangeRestart OrderBookEventType = "ExchangeRestart"
	Update          OrderBookEventType = "Update"
)

type OrderBookEvent struct {
	Type OrderBookEventType
	OrderBook
}
