package mocks

import (
	"testing"

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
