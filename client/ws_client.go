// Websocket client implementation. This will be used in tests.
package client

import (
	"fmt"
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/gorilla/websocket"
	"net/http"
)

// A websocket client subscribes and unsubscribes to events
type WSClient struct {
	host   string
	closed bool
	conn   *websocket.Conn
}

// create a new connection
func NewWSClient(addr string) *WSClient {
	return &WSClient{
		host: addr,
	}
}

func (this *WSClient) Dial() (*http.Response, error) {
	dialer := websocket.DefaultDialer
	rHeader := http.Header{}
	conn, r, err := dialer.Dial(this.host, rHeader)
	if err != nil {
		return r, err
	}
	this.conn = conn

	return r, nil
}

// returns a channel from which messages can be pulled
// from a go routine that reads the socket.
// if the ws returns an error (eg. closes), we return
func (this *WSClient) StartRead() <-chan []byte {
	ch := make(chan []byte)
	go func() {
		for {
			_, msg, err := this.conn.ReadMessage()
			if err != nil {
				if !this.closed {
					// TODO For now.
					fmt.Println("Error: " + err.Error())
					close(ch)
				}
				return
			}
			ch <- msg
		}
	}()
	return ch
}

func (this *WSClient) WriteMsg(msg []byte) {
	this.conn.WriteMessage(websocket.TextMessage, msg)
}

func (this *WSClient) Close() {
	this.closed = true
	this.conn.Close()
}
