package cryptotrade

import (
	"time"

	"github.com/sirupsen/logrus"
)

type event int

const (
	stop event = 1
)

type FishingStore struct {
	event chan event
}

func NewFishingStore() *FishingStore {
	var f FishingStore

	f.event = make(chan event, 10)

	return &f
}

func (store *FishingStore) Run() {

	logrus.Infoln("FishingStore is starting")

	store.onStart()

	logrus.Infoln("FishingStore starts successfully")

	for {
		select {
		case event := <-store.event:
			if event == stop {

				logrus.Infoln("FishingStore is stopping")

				store.onStop()

				logrus.Infoln("FishingStore stopped")

				return
			}
		default:
			time.Sleep(time.Second)
			logrus.Infoln("do nothing")
		}
	}

}

func (store *FishingStore) Stop() {
	store.event <- stop
}

func (*FishingStore) onStart() {
	//dummy := NewDummySupervisor()
}

func (*FishingStore) onStop() {

}
