package entity

import (
	"time"
)

type BuyOrderEventType string

const (
	None       BuyOrderEventType = "None"
	FillOrKill BuyOrderEventType = "FillOrKill"
)

type BuyOrderEvent struct {
	Type       BuyOrderEventType
	CreateDate time.Time
	OrderBook
}
