package websocketproxy

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/imdario/mergo"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Config struct {
	URL url.URL
}

var Default Config = Config{
	URL: url.URL{
		Scheme: "wss",
		Host:   "...example.com",
	},
}

type command string

const (
	connect    command = "connect"
	disconnect command = "disconnect"
)

type event struct {
	command    command
	callbackFn func(err error)
}

func (e *event) callback(err error) {
	if e.callbackFn != nil {
		e.callbackFn(err)
	}
}

type state string

const (
	connecting    state = "connecting"
	connected     state = "connected"
	disconnecting state = "disconnecting"
	disconnected  state = "disconnected"
)

type ErrorType string

const (
	ConnectionError    ErrorType = "ConnectionError"
	InvalideStateError ErrorType = "InvalideStateError"
)

type Error struct {
	Type ErrorType
	Err  error
}

func newError(t ErrorType, e error) *Error {
	return &Error{t, e}
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%v - %v]", e.Type, e.Err)
}

type WebSocketProxy struct {
	config Config
	stopWg sync.WaitGroup
	state  state
	conn   *websocket.Conn
	events chan *event
	errors chan *Error
}

func New(c *Config) *WebSocketProxy {

	ws := &WebSocketProxy{
		config: Default,
		state:  disconnected,
		events: make(chan *event, 10),
		errors: make(chan *Error, 10),
	}

	if c != nil {
		if err := mergo.Merge(&ws.config, c, mergo.WithOverride); err != nil {
			logger.Fatalf("Merge default config failed. err: [%v]", err)
		}
	}

	logger.Debugf("Use config [%#v]", ws.config)

	ws.stopWg.Add(1)
	go func() {
		defer ws.stopWg.Done()
		ws.worker()
	}()

	return ws
}

func (this *WebSocketProxy) Stop(ctx context.Context) error {
	logger.Infof("Stopping.")

	wait := make(chan int, 1)
	go func() {
		close(this.events)
		this.stopWg.Wait()
		if this.conn != nil {
			err := this.conn.Close()
			if err != nil {
				logger.Warnf("Disconnect to [%v] failed. err: [%v]", this.config.URL.String(), err)
			}
		}
		close(wait)
	}()

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			logger.Warnf("Stopped with an error [%v].", err)
			return err
		}
	case <-wait:
	}

	logger.Infof("Stopped.")
	return nil
}

func (this *WebSocketProxy) Error() error {
	select {
	case err := <-this.errors:
		return err
	default:
		return nil
	}
}

func (this *WebSocketProxy) Connect() error {

	c := make(chan error, 1)

	fn := func(e error) {
		c <- e
	}

	this.events <- &event{
		command:    connect,
		callbackFn: fn,
	}

	err := <-c

	return err
}

func (this *WebSocketProxy) Disconnect() error {

	c := make(chan error, 1)

	fn := func(e error) {
		c <- e
	}

	this.events <- &event{
		command:    disconnect,
		callbackFn: fn,
	}

	err := <-c

	return err
}

func (this *WebSocketProxy) worker() {

	defer func() {
		if this.conn != nil {
			logger.Debugf("Connection is closing")
			this.conn.Close()
			logger.Debugf("Connection closed")
		}
	}()

	for {
		e, ok := <-this.events
		if !ok {
			logger.Debugf("Stop worker since event queue is closed.")
			return
		}

		switch e.command {
		case connect:
			if this.state != disconnected {
				err := fmt.Errorf("Web socket allready connected")
				logger.Warnf("%v.", err)
				e.callback(newError(InvalideStateError, err))
				continue
			}

			this.state = connecting
			logger.Infof("Create connection to [%v].", this.config.URL.String())

			conn, _, err := websocket.DefaultDialer.Dial(this.config.URL.String(), nil)
			if err != nil {
				logger.Warnf("Connect to [%v] failed. err: [%v]", this.config.URL.String(), err)
				this.errors <- newError(ConnectionError, err)
				e.callback(newError(ConnectionError, err))
				this.state = disconnected
				continue
			}

			this.conn = conn
			logger.Infof("Connection to [%v] is established.", this.config.URL.String())
			this.state = connected
			e.callback(nil)

		case disconnect:
			if this.state != connected {
				err := fmt.Errorf("Web socket dose not connected")
				logger.Warnf("%v.", err)
				e.callback(newError(InvalideStateError, err))
				continue
			}

			this.state = disconnecting
			logger.Infof("Connection is closing")

			err := this.conn.Close()
			if err != nil {
				logger.Warnf("Disconnect to [%v] failed. err: [%v]", this.config.URL.String(), err)
				this.errors <- newError(ConnectionError, err)
				e.callback(newError(ConnectionError, err))
				this.state = disconnected
				continue
			}

			this.conn = nil

			logger.Infof("Connection closed")
			this.state = disconnected
			e.callback(nil)

		default:
		}
	}
}
