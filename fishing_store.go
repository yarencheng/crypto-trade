package cryptotrade

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type Event int

const (
	stop Event = 1
)

type FishingStore struct {
	event chan Event
}

func NewFishingStore() *FishingStore {
	var f FishingStore

	f.event = make(chan Event, 10)

	return &f
}

func (this *FishingStore) Run() {

	logrus.Infoln("FishingStore starts")

	for {
		select {
		case event := <-this.event:
			fmt.Println(event)
			if event == stop {
				logrus.Infoln("stopping")
				break
			}
		default:
			time.Sleep(time.Second)
			logrus.Infoln("do nothing")
		}
	}

	logrus.Infoln("FishingStore stopped")
}

func (this *FishingStore) Stop() {
	this.event <- stop

}
