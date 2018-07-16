package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type eventType string

const (
	send        eventType = "send"
	receive     eventType = "receive"
	sendJSON    eventType = "sendJSON"
	receiveJSON eventType = "receiveJSON"
	sendPing    eventType = "sendPing"
	receivePing eventType = "receivePing"
	sendPong    eventType = "sendPong"
	receivePong eventType = "receivePong"
	delay       eventType = "delay"
)

type event struct {
	tvpe    eventType
	message string
	delay   time.Duration
}

type WebSocketMock struct {
	server          *httptest.Server
	events          []event
	assertErr       []error
	clientCount     int
	clientCountLock sync.Mutex
	done            chan int
}

func NewWebSocketMock() *WebSocketMock {

	mock := &WebSocketMock{
		events:    make([]event, 0),
		assertErr: make([]error, 0),
		done:      make(chan int, 1),
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handler))

	return mock
}

func (mock *WebSocketMock) Close() {
	mock.server.Close()
}

func (mock *WebSocketMock) Assert(t *testing.T, ctxs ...context.Context) {

	t.Helper()

	{
		mock.clientCountLock.Lock()
		if len(mock.events) == 0 {
			mock.clientCountLock.Unlock()
			return
		}
		mock.clientCountLock.Unlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if len(ctxs) != 0 {
		ctx = ctxs[0]
	}

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			t.Errorf("WebSocketMock didn't receive a incoming connection. err: [%v]", err)
			t.Fail()
		}
	case <-mock.done:
	}

	for _, e := range mock.assertErr {
		t.Error(e)
	}
}

func (mock *WebSocketMock) URL() *url.URL {
	u, err := url.Parse(mock.server.URL)
	if err != nil {
		panic(err)
	}
	u.Scheme = "ws"
	return u
}

func (mock *WebSocketMock) Send(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    send,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) Receive(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    receive,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) SendJSON(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    sendJSON,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) ReceiveJSON(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    receiveJSON,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) SendPing(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    sendPing,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) ReceivePing(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    receivePing,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) SendPong(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    sendPong,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) ReceivePong(messages ...string) *WebSocketMock {
	for _, m := range messages {
		mock.events = append(mock.events, event{
			tvpe:    receivePong,
			message: m,
		})
	}
	return mock
}

func (mock *WebSocketMock) Delay(delays ...time.Duration) *WebSocketMock {
	for _, d := range delays {
		mock.events = append(mock.events, event{
			tvpe:  delay,
			delay: d,
		})
	}
	return mock
}

func (mock *WebSocketMock) handler(w http.ResponseWriter, r *http.Request) {
	defer close(mock.done)
	{
		mock.clientCountLock.Lock()
		mock.clientCount++
		if mock.clientCount == 2 {
			mock.assertErr = append(mock.assertErr, fmt.Errorf("This mock only support one connection."))
		}
		if mock.clientCount >= 2 {
			mock.clientCountLock.Unlock()
			return
		}
		mock.clientCountLock.Unlock()
	}

	upgrader := websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s := fmt.Sprintf("Create web socket server failed. err: [%v]", err)
		panic(s)
	}
	defer c.Close()
	defer c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "See you!"))

	pings := make(chan string, 1)
	defer close(pings)
	c.SetPingHandler(func(message string) error {
		pings <- message
		return nil
	})

	pongs := make(chan string, 1)
	defer close(pongs)
	c.SetPongHandler(func(message string) error {
		pongs <- message
		return nil
	})

	reads := make(chan interface{}, 1) // string, or error
	go func() {
		for {
			_, m, err := c.ReadMessage()
			if err != nil {
				if _, ok := err.(*websocket.CloseError); ok {
					return
				} else {
					reads <- fmt.Errorf("Read a message from client with an error: [%v].", err)
				}
			} else {
				reads <- string(m)
			}
		}
	}()

	for _, event := range mock.events {
		switch event.tvpe {
		case send:
			err := c.WriteMessage(websocket.TextMessage, []byte(event.message))
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Send message [%v] failed with an error: [%v].", event.message, err))
				return
			}
		case sendJSON:
			var j interface{}
			err := json.NewDecoder(strings.NewReader(event.message)).Decode(&j)
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Decode message [%v] to json failed with an error: [%v].", event.message, err))
				return
			}
			err = c.WriteJSON(j)
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Send message [%v] failed with an error: [%v].", event.message, err))
				return
			}
		case receive:
			m := <-reads
			if err, ok := m.(error); ok {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Read a message from client with an error: [%v].", err))
				return
			}
			if m != event.message {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Expect get [%v] but [%v]", event.message, m))
				return
			}
		case receiveJSON:
			var expected interface{}
			err := json.NewDecoder(strings.NewReader(event.message)).Decode(&expected)
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Decode message [%v] to json failed with an error: [%v].", event.message, err))
				return
			}

			m := <-reads
			if err, ok := m.(error); ok {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Read a message from client with an error: [%v].", err))
				return
			}

			var actual interface{}
			err = json.NewDecoder(strings.NewReader(m.(string))).Decode(&actual)
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Decode message [%v] to json failed with an error: [%v].", m.(string), err))
				return
			}

			if !reflect.DeepEqual(expected, actual) {
				b, _ := json.Marshal(actual)
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Expect get JSON [%v] but [%v]", event.message, string(b)))
				return
			}
		case sendPing:

			err := c.WriteMessage(websocket.PingMessage, []byte(event.message))
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Send ping [%v] failed with an error: [%v].", event.message, err))
				return
			}

		case receivePing:

			m, ok := <-pings
			if !ok {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Read a ping failed since channel is closed"))
				return
			}

			if m != event.message {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Expect get ping [%v] but [%v]", event.message, m))
				return
			}

		case sendPong:

			err := c.WriteMessage(websocket.PongMessage, []byte(event.message))
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Send pong [%v] failed with an error: [%v].", event.message, err))
				return
			}

		case receivePong:

			m, ok := <-pongs
			if !ok {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Read a pong failed since channel is closed"))
				return
			}

			if m != event.message {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Expect get pong [%v] but [%v]", event.message, m))
				return
			}

		case delay:

			time.Sleep(event.delay)

		default:
			s := fmt.Sprintf("Unknown tvpe: [%v]", event.tvpe)
			panic(s)
		}
	}

	time.Now()

}
