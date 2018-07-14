package websocket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tidwall/gjson"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"
)

type WebSocketTestSuite struct {
	suite.Suite
	server *httptest.Server
	url    *url.URL
	in     chan gjson.Result
}

func TestWebSocketTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &WebSocketTestSuite{})
}

func (s *WebSocketTestSuite) SetupTest() {

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		s.in = make(chan gjson.Result, 10)

		go func() {
			for {
				j, ok := <-s.in
				if !ok {
					break
				}
				err := c.WriteJSON(j.String())
				s.NoError(err)
			}
		}()

		for {
			_, message, err := c.ReadMessage()
			fmt.Printf("message: [%v], err: [%v]\n", string(message), err)
		}
	}))

	u, err := url.Parse(s.server.URL)
	s.Require().NoError(err)
	u.Scheme = "ws"

	s.url = u
}

func (s *WebSocketTestSuite) TearDownTest() {
	close(s.in)
	s.server.Close()
}

func (s *WebSocketTestSuite) TestStart() {
	// arrange
	ws := New(&Config{
		URL:  *s.url,
		Name: "WebSocketTestSuite",
		Reconnect: Reconnect{
			OnClosed:      "false",
			OnConnectFail: "false",
		},
	})

	// action
	err := ws.Start()
	s.Require().NoError(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		s.Require().NoError(ws.Stop(ctx))
	}()
}

func (s *WebSocketTestSuite) TestCloseEvent() {
	// arrange
	ws := New(&Config{
		URL:  *s.url,
		Name: "WebSocketTestSuite",
		Reconnect: Reconnect{
			OnClosed:      "false",
			OnConnectFail: "false",
		},
	})

	// action
	err := ws.Start()
	s.Require().NoError(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		s.Require().NoError(ws.Stop(ctx))
	}()

	time.Sleep(5 * time.Second)
}
