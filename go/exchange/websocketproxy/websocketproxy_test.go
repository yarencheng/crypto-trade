package websocketproxy

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

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

func Test_SetConnectedHandler(t *testing.T) {
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
	called := make(chan int, 1)
	ws.SetConnectedHandler(func(in <-chan *gjson.Result, out chan<- *gjson.Result) {
		close(called)
	})
	err := ws.Connect()
	require.NoError(t, err)

	// assert
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
	case <-called:
	}
	assert.NoError(t, ctx.Err())
}

func Test_SetPingFailedHandler(t *testing.T) {
	t.Parallel()

	// arrange: mock for web socket server
	wsMock := mocks.NewWebSocketMock()
	defer wsMock.Close()
	wsMock.ReceivePing("ping")
	wsMock.Delay(1 * time.Second)
	wsMock.SendPong("pong")

	// arrange: web socket proxy
	ws := New(&Config{
		URL:          *wsMock.URL(),
		PingInterval: time.Millisecond * 100,
	})
	defer ws.Stop(context.Background())

	// action
	called := make(chan int, 1)
	ws.SetPingFailedHandler(func(delay time.Duration) {
		close(called)
	})
	err := ws.Connect()
	require.NoError(t, err)

	// assert
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		assert.Fail(t, "timeout")
	case <-called:
		// pass
	}
}
