package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/exchanges"
	"github.com/yarencheng/crypto-trade/traders"
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

	trader := traders.NewSimpleTrader()

	trader.AddExchange(exchanges.NewDummyExchange())

	logrus.Infoln("Start", trader)
	if e := trader.Start(); e != nil {
		logrus.Infoln("Failed to start", trader, ". Caused by: ", e)
		os.Exit(1)
	}

	<-stop

	logrus.Infoln("Stop", trader)
	if e := trader.Stop(); e != nil {
		logrus.Infoln("Failed to stop", trader, ". Caused by: ", e)
		os.Exit(1)
	}
}
