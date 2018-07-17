package websocketproxy

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
	PingInterval time.Duration
	BufferSize   int
}

var Default Config = Config{
	URL: url.URL{
		Scheme: "wss",
		Host:   "...example.com",
	},
	PingInterval: 5 * time.Second,
	BufferSize:   100,
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
	config             Config
	readStop           chan int
	readWg             sync.WaitGroup
	pingStop           chan int
	pingWg             sync.WaitGroup
	workerWg           sync.WaitGroup
	state              state
	conn               *websocket.Conn
	events             chan *event
	in                 chan *gjson.Result
	out                chan *gjson.Result
	connectedFn        func(in <-chan *gjson.Result, out chan<- *gjson.Result)
	connectedFnLock    sync.Mutex
	disconnectedFn     func(code int, message string)
	disconnectedFnLock sync.Mutex
	pingFailedFn       func(delay time.Duration)
	pingFailedFnLock   sync.Mutex
}

func New(c *Config) *WebSocketProxy {

	ws := &WebSocketProxy{
		config: Default,
		state:  disconnected,
		events: make(chan *event, 10),
	}

	if c != nil {
		if err := mergo.Merge(&ws.config, c, mergo.WithOverride); err != nil {
			logger.Fatalf("Merge default config failed. err: [%v]", err)
		}
	}

	logger.Debugf("Use config [%#v]", ws.config)

	ws.workerWg.Add(1)
	go func() {
		defer ws.workerWg.Done()
		ws.worker()
	}()

	return ws
}

func (this *WebSocketProxy) Stop(ctx context.Context) error {
	logger.Infof("Stopping.")

	wait := make(chan int, 1)
	go func() {
		close(this.events)
		this.workerWg.Wait()
		if this.state == connected {
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

func (this *WebSocketProxy) SetConnectedHandler(fn func(in <-chan *gjson.Result, out chan<- *gjson.Result)) {
	this.connectedFnLock.Lock()
	defer this.connectedFnLock.Unlock()
	this.connectedFn = fn
}

func (this *WebSocketProxy) SetPingFailedHandler(fn func(delay time.Duration)) {
	this.pingFailedFnLock.Lock()
	defer this.pingFailedFnLock.Unlock()
	this.pingFailedFn = fn
}

func (this *WebSocketProxy) SetDisconnectedHandler(fn func(code int, message string)) {
	this.disconnectedFnLock.Lock()
	defer this.disconnectedFnLock.Unlock()
	this.disconnectedFn = fn
}

func (this *WebSocketProxy) worker() {

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

			logger.Infof("Create connection to [%v].", this.config.URL.String())
			this.state = connecting

			conn, _, err := websocket.DefaultDialer.Dial(this.config.URL.String(), nil)
			if err != nil {
				logger.Warnf("Connect to [%v] failed. err: [%v]", this.config.URL.String(), err)
				e.callback(newError(ConnectionError, err))
				this.state = disconnected
				continue
			}

			logger.Infof("Connection to [%v] is established.", this.config.URL.String())
			this.conn = conn
			this.state = connected
			e.callback(nil)

			this.pingStop = make(chan int, 1)
			this.pingWg.Add(1)
			go func() {
				defer this.pingWg.Done()
				this.pingWorker()
			}()

			this.readStop = make(chan int, 1)
			this.readWg.Add(1)
			this.in = make(chan *gjson.Result, this.config.BufferSize)
			go func() {
				defer this.readWg.Done()

				this.readWorker()
			}()

			this.connectedFnLock.Lock()
			if this.connectedFn != nil {
				this.connectedFn(this.in, nil)
			}
			this.connectedFnLock.Unlock()

		case disconnect:
			if this.state != connected {
				err := fmt.Errorf("Web socket dose not connected")
				logger.Warnf("%v.", err)
				e.callback(newError(InvalideStateError, err))
				continue
			}

			logger.Infof("Connection is closing")
			this.state = disconnecting

			close(this.pingStop)
			close(this.readStop)
			this.pingWg.Wait()
			this.readWg.Wait()

			err := this.conn.Close()
			if err != nil {
				logger.Warnf("Disconnect to [%v] failed. err: [%v]", this.config.URL.String(), err)
				e.callback(newError(ConnectionError, err))
				this.state = disconnected
				continue
			}

			logger.Infof("Connection closed")
			this.conn = nil
			this.state = disconnected
			e.callback(nil)

		default:
		}
	}
}

func (this *WebSocketProxy) pingWorker() {

	received := make(chan int, 1)
	defer close(received)

	this.conn.SetPongHandler(func(data string) error {
		received <- 1
		return nil
	})
	defer this.conn.SetPongHandler(nil)

	timep := func() *time.Time {
		t := time.Now()
		return &t
	}

	received <- 1
	tLock := &sync.Mutex{}
	t := timep()

	for {
		select {
		case <-this.pingStop:
			return
		case <-received:

			tLock.Lock()
			start := *t
			tLock.Unlock()

			end := time.Now()
			logger.Debugf("Receives a pong in [%v] seconds", end.Sub(start).Seconds())

			delay := end.Sub(start)
			if delay < this.config.PingInterval {
				time.Sleep(this.config.PingInterval - delay)
			}

			err := this.conn.WriteMessage(websocket.PingMessage, []byte("ping"))

			if err != nil {
				logger.Warnf("Send ping message failed. err: [%v]", err)
			}

			tLock.Lock()
			t = timep()
			tLock.Unlock()

			go func() {
				time.Sleep(2 * this.config.PingInterval)

				tLock.Lock()
				last := *t
				tLock.Unlock()

				delay_ := time.Now().Sub(last)
				if delay_ > 2*this.config.PingInterval {
					logger.Warnf("It took too long for a pong in [%v] seconds", delay_.Seconds())

					this.pingFailedFnLock.Lock()
					defer this.pingFailedFnLock.Unlock()

					if this.pingFailedFn != nil {
						this.pingFailedFn(delay_)
					}
				}
			}()
		}
	}
}

func (this *WebSocketProxy) readWorker() {

	done := make(chan int, 1)

	go func() {

		defer close(this.in)
		defer close(done)

		for {
			var j interface{}
			err := this.conn.ReadJSON(&j)

			if err != nil {
				if we, ok := err.(*websocket.CloseError); ok {
					switch we.Code {
					case websocket.CloseNormalClosure:
						logger.Infof("Remote server is closed(code=%v) with message [%v].", we.Code, we.Text)
					default:
						logger.Warnf("Remote server is closed(code=%v) with message [%v].", we.Code, we.Text)
					}

					this.disconnectedFnLock.Lock()
					defer this.disconnectedFnLock.Unlock()
					if this.disconnectedFn != nil {
						this.disconnectedFn(we.Code, we.Text)
					}

				} else {
					logger.Warnf("Read JSON failed. err: [%v].", err)
				}

				return
			}

			b, err := json.Marshal(j)
			if err != nil {
				logger.Warnf("Failed to marshal json [%#v]. err: [%v]", j, err)
				return
			}

			gj := gjson.ParseBytes(b)

			logger.Debugf("Receive [%v]", gj.String())

			this.in <- &gj
		}
	}()

	select {
	case <-this.readStop:
	case <-done:
	}
}
