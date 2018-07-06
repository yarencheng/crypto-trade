package data

import (
	"encoding/json"
)

type Order struct {
	From Currency
	To   Currency
}

func (o Order) String() string {
	j, _ := json.Marshal(o)
	return string(j)
}
