package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/tidwall/gjson"

	"github.com/stretchr/testify/assert"

	"github.com/yarencheng/crypto-trade/go/mocks"

	"github.com/stretchr/testify/require"
)

func Test_Start(t *testing.T) {
	t.Parallel()

	// arrange
	wsMock := mocks.NewWebSocketMock()
	defer wsMock.Close()
	ws := New(&Config{
		URL: *wsMock.URL(),
		Reconnect: Reconnect{
			OnClosed:      "false",
			OnConnectFail: "false",
		},
	})

	// action
	err := ws.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		require.NoError(t, ws.Stop(ctx))
	}()

	// assert
	assert.Equal(t, 0, ws.ConnectFailedCount())
}

func Test_Close(t *testing.T) {
	t.Parallel()

	// arrange
	wsMock := mocks.NewWebSocketMock()
	defer wsMock.Close()
	closed := make(chan int, 1)
	ws := New(&Config{
		URL: *wsMock.URL(),
		Reconnect: Reconnect{
			OnClosed:      "false",
			OnConnectFail: "false",
		},
		EventHandler: EventHandler{
			OnClosed: func() {
				close(closed)
			},
		},
	})

	// action
	err := ws.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		require.NoError(t, ws.Stop(ctx))
	}()

	// assert
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	select {
	case <-ctx.Done():
	case <-closed:
	}

	assert.Equal(t, 1, ws.ClosedCount())
	assert.NoError(t, ctx.Err())
}

func Test_Read(t *testing.T) {
	t.Parallel()

	// arrange
	wsMock := mocks.NewWebSocketMock()
	defer wsMock.Close()
	wsMock.Send(`{"aa":"AA"}`)
	ws := New(&Config{
		URL: *wsMock.URL(),
		Reconnect: Reconnect{
			OnClosed:      "false",
			OnConnectFail: "false",
		},
	})

	// action
	err := ws.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		require.NoError(t, ws.Stop(ctx))
	}()

	// assert
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	var gj *gjson.Result
	defer cancel()
	select {
	case <-ctx.Done():
	case gj = <-ws.Read():
	}

	require.NoError(t, ctx.Err())
	assert.Equal(t, gjson.Parse(`{"aa":"AA"}`), *gj)
}
