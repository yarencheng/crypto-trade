package mocks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

type eventType string

const (
	send        eventType = "send"
	receive     eventType = "receive"
	sendJSON    eventType = "sendJSON"
	receiveJSON eventType = "receiveJSON"
)

type event struct {
	tvpe    eventType
	message string
}

type WebSocketMock struct {
	server          *httptest.Server
	events          []event
	assertErr       []error
	clientCount     int
	clientCountLock sync.Mutex
}

func NewWebSocketMock() *WebSocketMock {

	mock := &WebSocketMock{
		events:    make([]event, 0),
		assertErr: make([]error, 0),
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handler))

	return mock
}

func (mock *WebSocketMock) Close() {
	mock.server.Close()
}

func (mock *WebSocketMock) Assert(t *testing.T) {
	t.Helper()

	mock.clientCountLock.Lock()
	defer mock.clientCountLock.Unlock()
	if mock.clientCount == 0 && len(mock.events) > 0 {
		t.Error("No any incoming connection")
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

func (mock *WebSocketMock) handler(w http.ResponseWriter, r *http.Request) {

	{
		mock.clientCountLock.Lock()
		defer mock.clientCountLock.Unlock()
		mock.clientCount++
		if mock.clientCount == 2 {
			mock.assertErr = append(mock.assertErr, fmt.Errorf("This mock only support one connection."))
		}
		if mock.clientCount >= 2 {
			return
		}
	}

	upgrader := websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s := fmt.Sprintf("Create web socket server failed. err: [%v]", err)
		panic(s)
	}
	defer c.Close()
	defer c.WriteMessage(websocket.CloseMessage, []byte("See you!"))

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
			_, m, err := c.ReadMessage()
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Read a message from client with an error: [%v].", err))
				return
			}
			if string(m) != event.message {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Expect get [%v] but [%v]", event.message, string(m)))
				return
			}
		case receiveJSON:
			var expected interface{}
			err := json.NewDecoder(strings.NewReader(event.message)).Decode(&expected)
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Decode message [%v] to json failed with an error: [%v].", event.message, err))
				return
			}
			var actual interface{}
			err = c.ReadJSON(&actual)
			if err != nil {
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Read a JSON from client with an error: [%v].", err))
				return
			}
			if !reflect.DeepEqual(expected, actual) {
				b, _ := json.Marshal(actual)
				mock.assertErr = append(mock.assertErr, fmt.Errorf("Expect get JSON [%v] but [%v]", event.message, string(b)))
				return
			}
		default:
			s := fmt.Sprintf("Unknown tvpe: [%v]", event.tvpe)
			panic(s)
		}
	}

}
