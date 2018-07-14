package bitfinex

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yarencheng/crypto-trade/go/entity"
)

func Test_Start(t *testing.T) {

	// arrange
	b := New()
	b.Currencies = []entity.Currency{entity.BTC, entity.ETH}
	orderBooks := make(chan entity.OrderBookEvent)
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

	time.Sleep(10 * time.Second)
}

type I interface {
	ff()
}

type A struct {
	a int
}

func (a *A) InitA() {
	fmt.Println("A.InitA()")
	a.a = 1
	a.Fn()
}

func (a *A) Fn() {
	fmt.Println("A.Fn()")
	a.a = 2
}

type B struct {
	A
	I
	a int
}

func (b *B) Fn() {
	fmt.Println("B.Fn()")
	b.a = 222
}

func Test_Startssss(t *testing.T) {
	b := &B{}
	b.InitA()

	fmt.Println(b.a)

	b.Fn()

	fmt.Println(b.a)

	b.ff()

	t.Fail()
}
