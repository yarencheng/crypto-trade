package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yarencheng/crypto-trade/go/mocks"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WebSocketTestSuite struct {
	suite.Suite
	wsMock *mocks.WebSocketMock
}

func TestWebSocketTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &WebSocketTestSuite{})
}

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
	ws := New(&Config{
		URL: *wsMock.URL(),
		Reconnect: Reconnect{
			OnClosed:      "false",
			OnConnectFail: "false",
		},
	})

	// time.Sleep(time.Second * 2)

	// wsMock.Close()

	// action
	err := ws.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		require.NoError(t, ws.Stop(ctx))
	}()

	time.Sleep(time.Second * 2)
	wsMock.Close()
	time.Sleep(time.Second * 2)

	assert.Equal(t, 1, ws.ClosedCount())
}
