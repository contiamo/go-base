package middlewares

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	server "github.com/contiamo/go-base/v3/pkg/http"
)

var upgrader = websocket.Upgrader{}

func echoWS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func createServer(opts []server.Option) (*http.Server, error) {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/ws/", echoWS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panic" {
			panic("PANIC!!!")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, r.Body)
	})

	return server.New(&server.Config{
		Handler: mux,
		Options: opts,
	})
}

func testWebsocketEcho(server string) error {
	u := "ws" + strings.TrimPrefix(server, "http")

	// Connect to the server
	ws, resp, err := websocket.DefaultDialer.Dial(u+"/ws/echo", nil)
	if err != nil {
		return err
	}
	defer ws.Close()
	defer resp.Body.Close()

	// Send message to server, read response and check to see if it's what we expect.
	err = ws.WriteMessage(websocket.TextMessage, []byte("hello"))
	if err != nil {
		return err
	}

	_, p, err := ws.ReadMessage()
	if err != nil {
		return err
	}

	if string(p) != "hello" {
		return fmt.Errorf("websocket echo expected \"hello\" but got \"%s\"", string(p))
	}

	return nil
}
