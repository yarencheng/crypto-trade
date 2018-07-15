package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/gorilla/websocket"
	"github.com/imdario/mergo"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Config struct {
	URL          url.URL
	Name         string
	Reconnect    Reconnect
	EventHandler EventHandler
}

type Reconnect struct {
	OnClosed      string
	OnConnectFail string
	OnReadError   string
	OnWriteError  string
}

type EventHandler struct {
	OnClosed      func()
	OnConnect     func()
	OnConnectFail func()
	OnReadError   func()
	OnWriteError  func()
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
	EventHandler: EventHandler{
		OnClosed:      func() {},
		OnConnect:     func() {},
		OnConnectFail: func() {},
		OnReadError:   func() {},
		OnWriteError:  func() {},
	},
}

type WebSocket struct {
	config             Config
	stop               chan int
	stopWg             sync.WaitGroup
	closedCount        int
	connectFailedCount int
	readErrorCount     int
	writeErrorCount    int
	readChan           chan *gjson.Result
	writeChan          chan *gjson.Result
}

func New(c *Config) *WebSocket {

	ws := &WebSocket{
		config:    Default,
		stop:      make(chan int, 1),
		readChan:  make(chan *gjson.Result, 100),
		writeChan: make(chan *gjson.Result, 100),
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

				ws.connectFailedCount++
				ws.config.EventHandler.OnConnectFail()

				if ws.config.Reconnect.OnConnectFail == "true" {
					logger.Infof("[%v] Reconnect after 1 second", ws.config.Name)
					time.Sleep(time.Second)
					continue
				} else {
					return
				}
			}

			logger.Infof("[%v] Connected to [%v]", ws.config.Name, ws.config.URL.String())
			ws.config.EventHandler.OnConnect()

			socketClose := make(chan int, 1)
			client.SetCloseHandler(func(code int, message string) error {
				logger.Infof("[%v] Socket closed: code=[%v] message=[%v]", ws.config.Name, code, message)
				close(socketClose)
				return nil
			})

			wg := sync.WaitGroup{}

			readError := make(chan int, 1)
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					var o interface{}
					err = client.ReadJSON(&o)
					if err != nil {
						if we, ok := err.(*websocket.CloseError); ok {
							logger.Infof("[%v] read a closing message: [%v:%v]", ws.config.Name, we.Code, we.Text)
						} else {
							logger.Errorf("[%v] read JSON failed. err [%v]", ws.config.Name, err)
							close(readError)
						}
						return
					}
					logger.Debugf("[%v] read JSON: [%#v]", ws.config.Name, o)

					j, err := json.Marshal(o)
					if err != nil {
						logger.Errorf("[%v] Failed to marshal JSON [%v]. err: [%v]", ws.config.Name, o, err)
						close(readError)
					}

					gj := gjson.ParseBytes(j)
					ws.readChan <- &gj
				}
			}()

			writeError := make(chan int, 1)
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					gj, ok := <-ws.writeChan
					if !ok {
						return
					}

					logger.Debugf("[%v] read JSON: [%v]", ws.config.Name, gj.String())

					err := client.WriteJSON(gj.Value())
					if err != nil {
						logger.Errorf("[%v] Failed to write JSON [%v]. err: [%v]", ws.config.Name, gj.String(), err)
						close(writeError)
						return
					}
				}
			}()

			select {
			case <-ws.stop:
				err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "See you!"))
				if err != nil {
					logger.Warnf("[%v] Failed to send closing message. err: [%v]", ws.config.Name, err)
					return
				}

				err = client.Close()
				if err != nil {
					logger.Errorf("[%v] Failed to close socket. err: [%v]", ws.config.Name, err)
					return
				}

				return
			case <-socketClose:
				err = client.Close()
				if err != nil {
					logger.Errorf("[%v] Failed to close socket. err: [%v]", ws.config.Name, err)
					return
				}

				ws.closedCount++
				ws.config.EventHandler.OnClosed()

				if ws.config.Reconnect.OnClosed != "true" {
					return
				}
			case <-readError:
				err = client.Close()
				if err != nil {
					logger.Errorf("[%v] Failed to close socket. err: [%v]", ws.config.Name, err)
					return
				}

				ws.readErrorCount++
				ws.config.EventHandler.OnReadError()

				if ws.config.Reconnect.OnReadError != "true" {
					return
				}
			case <-writeError:
				err = client.Close()
				if err != nil {
					logger.Errorf("[%v] Failed to close socket. err: [%v]", ws.config.Name, err)
					return
				}

				ws.writeErrorCount++
				ws.config.EventHandler.OnWriteError()

				if ws.config.Reconnect.OnWriteError != "true" {
					return
				}
			}

			close(ws.readChan)
			close(ws.writeChan)

			logger.Debugf("Wait read/write goroutine to finish")
			wg.Wait()

			logger.Infof("[%v] Reconnect after 1 second", ws.config.Name)
			time.Sleep(time.Second)

			ws.readChan = make(chan *gjson.Result, 100)
			ws.writeChan = make(chan *gjson.Result, 100)
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

func (ws *WebSocket) Read() (*gjson.Result, bool) {
	gj, ok := <-ws.readChan
	return gj, ok
}

func (ws *WebSocket) Write(gj *gjson.Result) bool {
	select {
	case ws.writeChan <- gj:
		return true
	default:
		return false
	}
}

func (ws *WebSocket) ClosedCount() int {
	return ws.closedCount
}

func (ws *WebSocket) ConnectFailedCount() int {
	return ws.connectFailedCount
}

func (ws *WebSocket) ReadErrorCount() int {
	return ws.readErrorCount
}

func (ws *WebSocket) WriteErrorCount() int {
	return ws.writeErrorCount
}
