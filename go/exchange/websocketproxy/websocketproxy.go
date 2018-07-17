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
	stop        command = "stop"
	connect     command = "connect"
	disconnect  command = "disconnect"
	pingTooLong command = "pingTooLong"
)

type event struct {
	command command
	ins     <-chan interface{}
	outs    chan<- interface{}
}

func (this *event) out(vs ...interface{}) {
	for _, v := range vs {
		this.outs <- v
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
	stop               chan int
	wg                 sync.WaitGroup
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
	pingTooLongFn      func(delay time.Duration)
	pingTooLongFnLock  sync.Mutex
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
		this.events <- &event{
			command: stop,
		}
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

	r := make(chan interface{}, 1)

	this.events <- &event{
		command: connect,
		outs:    r,
	}

	err := <-r

	if err != nil {
		return err.(error)
	}

	return nil
}

func (this *WebSocketProxy) Disconnect() error {

	r := make(chan interface{}, 1)

	this.events <- &event{
		command: disconnect,
		outs:    r,
	}

	err := <-r

	if err != nil {
		return err.(error)
	}

	return nil
}

func (this *WebSocketProxy) SetConnectedHandler(fn func(in <-chan *gjson.Result, out chan<- *gjson.Result)) {
	this.connectedFnLock.Lock()
	defer this.connectedFnLock.Unlock()
	this.connectedFn = fn
}

func (this *WebSocketProxy) SetPingTooLongFnHandler(fn func(delay time.Duration)) {
	this.pingTooLongFnLock.Lock()
	defer this.pingTooLongFnLock.Unlock()
	this.pingTooLongFn = fn
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
			logger.Errorf("Stop worker since event queue is closed.")
			return
		}

		switch e.command {
		case stop:
			logger.Debugf("Stop worker since event queue is closed.")
			return
		case connect:
			if this.state != disconnected {
				err := fmt.Errorf("Web socket allready connected")
				logger.Warnf("%v.", err)
				e.out(newError(InvalideStateError, err))
				continue
			}

			logger.Infof("Create connection to [%v].", this.config.URL.String())
			this.state = connecting

			conn, _, err := websocket.DefaultDialer.Dial(this.config.URL.String(), nil)
			if err != nil {
				logger.Warnf("Connect to [%v] failed. err: [%v]", this.config.URL.String(), err)
				e.out(newError(ConnectionError, err))
				this.state = disconnected
				continue
			}

			logger.Infof("Connection to [%v] is established.", this.config.URL.String())
			this.conn = conn
			this.state = connected
			e.out(nil)

			this.stop = make(chan int, 1)

			this.wg.Add(1)
			go func() {
				defer this.wg.Done()
				this.pingWorker()
			}()

			this.wg.Add(1)
			this.in = make(chan *gjson.Result, this.config.BufferSize)
			go func() {
				defer this.wg.Done()
				this.readWorker()
			}()

			this.wg.Add(1)
			this.out = make(chan *gjson.Result, this.config.BufferSize)
			go func() {
				defer this.wg.Done()
				this.writeWorker()
			}()

			this.connectedFnLock.Lock()
			if this.connectedFn != nil {
				this.connectedFn(this.in, this.out)
			} else {
				logger.Warnf("Connected handler is not set.")
			}
			this.connectedFnLock.Unlock()

		case disconnect:
			if this.state != connected {
				err := fmt.Errorf("Web socket dose not connected")
				logger.Warnf("%v.", err)
				e.out(newError(InvalideStateError, err))
				continue
			}

			logger.Infof("Connection is closing")
			this.state = disconnecting

			close(this.stop)
			this.wg.Wait()

			err := this.conn.Close()
			if err != nil {
				logger.Warnf("Disconnect to [%v] failed. err: [%v]", this.config.URL.String(), err)
				e.out(newError(ConnectionError, err))
				this.state = disconnected
				continue
			}

			logger.Infof("Connection closed")
			this.conn = nil
			this.state = disconnected
			e.out(nil)

			this.disconnectedFnLock.Lock()
			if this.disconnectedFn != nil {
				code := (<-e.ins).(int)
				text := (<-e.ins).(string)
				this.disconnectedFn(code, text)
			}
			this.disconnectedFnLock.Unlock()

		case pingTooLong:

			this.pingTooLongFnLock.Lock()
			if this.pingTooLongFn != nil {
				delay := (<-e.ins).(time.Duration)
				this.pingTooLongFn(delay)
			} else {
				logger.Warnf("PingTooLong handler is not set.")
			}
			this.pingTooLongFnLock.Unlock()

		default:
		}
	}

	time.Sleep(1) //debug
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
		case <-this.stop:
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

					in := make(chan interface{}, 1)
					in <- delay_
					close(in)

					this.events <- &event{
						command: pingTooLong,
						ins:     in,
						outs:    make(chan interface{}, 1),
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

					in := make(chan interface{}, 2)
					in <- we.Code
					in <- we.Text
					close(in)

					this.events <- &event{
						command: disconnect,
						ins:     in,
						outs:    make(chan interface{}, 1),
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
	case <-this.stop:
	case <-done:
	}
}

func (this *WebSocketProxy) writeWorker() {

	done := make(chan int, 1)

	go func() {

		defer close(this.out)
		defer close(done)

		for {

			gj := <-this.out

			err := this.conn.WriteMessage(websocket.TextMessage, []byte(gj.String()))

			if err != nil {
				logger.Warnf("Write JSON [%v] failed. err: [%v].", gj.String(), err)
				return
			}
		}
	}()

	select {
	case <-this.stop:
	case <-done:
	}
}
