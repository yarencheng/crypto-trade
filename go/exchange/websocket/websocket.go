package websocket

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/imdario/mergo"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Config struct {
	URL       url.URL
	Name      string
	Reconnect Reconnect
}

type Reconnect struct {
	OnClosed      string
	OnConnectFail string
	OnReadError   string
	OnWriteError  string
}

var Default Config = Config{
	URL: url.URL{
		Scheme: "wss",
		Host:   "example.com",
	},
	Name: "Give me a name",
	Reconnect: Reconnect{
		OnClosed:      "true",
		OnConnectFail: "true",
		OnReadError:   "false",
		OnWriteError:  "false",
	},
}

type WebSocket struct {
	config Config
	stop   chan int
	stopWg sync.WaitGroup
	client *websocket.Conn
}

func New(c *Config) *WebSocket {

	ws := &WebSocket{
		config: Default,
		stop:   make(chan int, 1),
	}

	if c != nil {
		if err := mergo.Merge(&ws.config, c, mergo.WithOverride); err != nil {
			logger.Fatalf("Merge default config failed. err: [%v]", err)
		}
	}

	logger.Debugf("[%v] Use config [%#v]", ws.config.Name, ws.config)

	return ws
}

func (ws *WebSocket) Start() error {
	logger.Infof("[%v] is starting.", ws.config.Name)

	ws.stopWg.Add(1)
	go func() {
		defer ws.stopWg.Done()

		for {

			logger.Infof("[%v] Connecting to [%v]", ws.config.Name, ws.config.URL.String())

			client, _, err := websocket.DefaultDialer.Dial(ws.config.URL.String(), nil)
			if err != nil {
				err = fmt.Errorf("[%v] Dial to [%v] failed. err: [%v]", ws.config.Name, ws.config.URL.String(), err)
				logger.Warnf("%v", err)
				if ws.config.Reconnect.OnConnectFail == "true" {
					logger.Infof("[%v] Reconnect after 1 second", ws.config.Name)
					time.Sleep(time.Second)
					continue
				} else {
					return
				}
			}
			ws.client = client

			socketClose := make(chan int, 1)
			client.SetCloseHandler(func(code int, message string) error {
				logger.Infof("[%v] Socket closed: code=[%v] message=[%v]", ws.config.Name, code, message)
				close(socketClose)
				return nil
			})

			select {
			case <-ws.stop:
				err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "See you!"))
				if err != nil {
					logger.Warnf("[%v] Failed to send closing message. err: [%v]", ws.config.Name, err)
				}
				err = client.Close()
				if err != nil {
					logger.Errorf("[%v] Failed to close socket. err: [%v]", ws.config.Name, err)
				}
				return
			case <-socketClose:
				err = client.Close()
				if err != nil {
					logger.Errorf("[%v] Failed to close socket. err: [%v]", ws.config.Name, err)
				}
				if ws.config.Reconnect.OnClosed == "true" {
					logger.Infof("[%v] Reconnect after 1 second", ws.config.Name)
					time.Sleep(time.Second)
					continue
				} else {
					return
				}
			}
		}

	}()

	logger.Infof("[%v] started.", ws.config.Name)

	return nil
}

func (ws *WebSocket) Stop(ctx context.Context) error {
	logger.Infof("[%v] is stopping.", ws.config.Name)

	wait := make(chan int, 1)

	go func() {
		close(ws.stop)
		ws.stopWg.Wait()
		wait <- 1
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait:
		logger.Infof("[%v] stopped.", ws.config.Name)
		return nil
	}
}
