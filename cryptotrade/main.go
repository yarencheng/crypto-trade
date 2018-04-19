package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	cryptotrade "github.com/yarencheng/crypto-trade"
)

func init() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}

var log = logrus.New()

func main() {

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	f := cryptotrade.NewFishingStore()

	go f.Run()

	<-stop

	f.Stop()
}
