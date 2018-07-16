package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gorilla/websocket"
)

func Test_WebSocketMock_doNothing(t *testing.T) {
	t.Parallel()

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// action
	mock.Assert(tMock)

	// assert
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_Receive_Pass(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.Receive("haha")

	// action
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	err = client.WriteMessage(websocket.TextMessage, []byte("haha"))
	require.NoError(t, err)
	err = client.Close()
	require.NoError(t, err)

	// action
	mock.Assert(tMock)

	// assert
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_Receive_Failed(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.Receive("hoho")

	// action
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	err = client.WriteMessage(websocket.TextMessage, []byte("haha"))
	require.NoError(t, err)
	err = client.Close()
	require.NoError(t, err)

	// action
	mock.Assert(tMock)

	// assert
	assert.True(t, tMock.Failed())
}

func Test_WebSocketMock_Send_Pass(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.Send("haha")

	// action
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)

	tm, m, err := client.ReadMessage()
	require.NoError(t, err)
	err = client.Close()
	require.NoError(t, err)

	// action
	mock.Assert(tMock)

	// assert
	assert.Equal(t, websocket.TextMessage, tm)
	assert.Equal(t, []byte("haha"), m)
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_ReceiveJSON_Pass(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceiveJSON(`{"aa": "AA"}`)

	// action
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	err = client.WriteJSON(map[string]string{
		"aa": "AA",
	})
	require.NoError(t, err)
	err = client.Close()
	require.NoError(t, err)

	// action
	mock.Assert(tMock)

	// assert
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_ReceiveJSON_Failed(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceiveJSON(`{"aa": "AA"}`)

	// action
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	err = client.WriteJSON(map[string]string{
		"bb": "BB",
	})
	require.NoError(t, err)
	err = client.Close()
	require.NoError(t, err)

	// action
	mock.Assert(tMock)

	// assert
	assert.True(t, tMock.Failed())
}

func Test_WebSocketMock_SendPing_OK(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.SendPing("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	message := make(chan string, 1)
	client.SetPingHandler(func(m string) error {
		message <- m
		return nil
	})

	// arrange: client must read message or ping handler will not be trigger
	go func() {
		for {
			_, _, err := client.ReadMessage()
			if err != nil {
				if _, ok := err.(*websocket.CloseError); !ok {
					require.NoError(t, err)
				} else {
					return
				}
			}
		}
	}()

	// assert
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		assert.Fail(t, "timeout")
	case m := <-message:
		assert.Equal(t, "haha", m)
	}

	// assert
	mock.Assert(tMock)
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_SendPong_OK(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.SendPong("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	message := make(chan string, 1)
	client.SetPongHandler(func(m string) error {
		message <- m
		return nil
	})

	// arrange: client must read message or pong handler will not be trigger
	go func() {
		for {
			_, _, err := client.ReadMessage()
			if err != nil {
				if _, ok := err.(*websocket.CloseError); !ok {
					require.NoError(t, err)
				} else {
					return
				}
			}
		}
	}()

	// assert
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		assert.Fail(t, "timeout")
	case m := <-message:
		assert.Equal(t, "haha", m)
	}

	// assert
	mock.Assert(tMock)
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_ReceivePing_OK(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceivePing("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	err = client.WriteMessage(websocket.PingMessage, []byte("haha"))
	require.NoError(t, err)

	// assert
	mock.Assert(tMock)
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_ReceivePing_messageAreNotSame(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceivePing("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	err = client.WriteMessage(websocket.PingMessage, []byte("yaya"))
	require.NoError(t, err)

	// assert
	mock.Assert(tMock)
	assert.True(t, tMock.Failed())
}

func Test_WebSocketMock_ReceivePing_getNothing(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceivePing("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	// assert
	mock.Assert(tMock)
	assert.True(t, tMock.Failed())
}

func Test_WebSocketMock_ReceivePong_OK(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceivePong("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	err = client.WriteMessage(websocket.PongMessage, []byte("haha"))
	require.NoError(t, err)

	// assert
	mock.Assert(tMock)
	assert.False(t, tMock.Failed())
}

func Test_WebSocketMock_ReceivePong_messageAreNotSame(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceivePong("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	err = client.WriteMessage(websocket.PongMessage, []byte("yaya"))
	require.NoError(t, err)

	// assert
	mock.Assert(tMock)
	assert.True(t, tMock.Failed())
}

func Test_WebSocketMock_ReceivePong_getNothing(t *testing.T) {
	t.Parallel()

	// arrange: mock testing.T
	tMock := new(testing.T)

	// arrange
	mock := NewWebSocketMock()
	defer mock.Close()
	mock.ReceivePong("haha") // action

	// arrange: create client
	client, _, err := websocket.DefaultDialer.Dial(mock.URL().String(), nil)
	require.NoError(t, err)
	defer client.Close()

	// assert
	mock.Assert(tMock)
	assert.True(t, tMock.Failed())
}
