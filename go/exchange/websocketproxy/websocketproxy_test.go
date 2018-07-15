package websocketproxy

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/yarencheng/crypto-trade/go/mocks"
)

func Test_Connect(t *testing.T) {
	t.Parallel()

	// arrange: mock for web socket server
	wsMock := mocks.NewWebSocketMock()
	defer wsMock.Close()

	// arrange: web socket proxy
	ws := New(&Config{
		URL: *wsMock.URL(),
	})
	defer ws.Stop(context.Background())

	// action
	err := ws.Connect()

	// assert
	assert.NoError(t, err)
}

func Test_Connect_ConnectionError(t *testing.T) {
	t.Parallel()

	// arrange: web socket proxy
	ws := New(&Config{
		URL: url.URL{
			Scheme: "ws",
			Host:   "...wrong..host..",
		},
	})
	defer ws.Stop(context.Background())

	// action
	err := ws.Connect()

	// assert
	assert.Error(t, err)
	assert.Equal(t, ConnectionError, err.(*Error).Type)
}

func Test_Connect_InvalideStateError(t *testing.T) {
	t.Parallel()

	// arrange: mock for web socket server
	wsMock := mocks.NewWebSocketMock()
	defer wsMock.Close()

	// arrange: web socket proxy
	ws := New(&Config{
		URL: *wsMock.URL(),
	})
	defer ws.Stop(context.Background())

	// action
	err := ws.Connect()
	require.NoError(t, err)
	err = ws.Connect() // connect again

	// assert
	assert.Error(t, err)
	assert.Equal(t, InvalideStateError, err.(*Error).Type)
}

func Test_Disconnect_InvalideStateError(t *testing.T) {
	t.Parallel()

	// arrange: mock for web socket server
	wsMock := mocks.NewWebSocketMock()
	defer wsMock.Close()

	// arrange: web socket proxy
	ws := New(&Config{
		URL: *wsMock.URL(),
	})
	defer ws.Stop(context.Background())

	// action
	err := ws.Disconnect()

	// assert
	assert.Error(t, err)
	assert.EqualValues(t, InvalideStateError, err.(*Error).Type)
}
