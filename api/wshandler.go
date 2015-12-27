package api

import (
	"github.com/gorilla/websocket"
	jww "github.com/spf13/jwalterweatherman"
	"net/http"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func (a *ApiHandlers) WsCombinedHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			jww.FATAL.Println(err)
		}
		return
	}

	go writer(ws, a)
	reader(ws)
}

func (a *ApiHandlers) WsFifteenSecHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			jww.FATAL.Println(err)
		}
		return
	}

	go writerFifteenSec(ws, a)
	reader(ws)
}

func (a *ApiHandlers) WsTenMinuteHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			jww.FATAL.Println(err)
		}
		return
	}

	go writerTenMin(ws, a)
	reader(ws)
}

// required PONG implementation
func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writer runs in a goroutine for each connected WS client. It emits all message returned by the observer.
func writer(ws *websocket.Conn, a *ApiHandlers) {
	pingTicker := time.NewTicker(pingPeriod)
	subscriber := a.getDBSubscriber()
	defer func() {
		subscriber.quitChan <- true
		pingTicker.Stop()
		ws.Close()
	}()

	for {
		select {
		case msg := <-subscriber.bufChan:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteJSON(msg); err != nil {
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// writerTenMin runs in a goroutine for each connected WS client, but only
// emits a message on the connected socket if the monitor returns a new
// TenMinute message.
func writerTenMin(ws *websocket.Conn, a *ApiHandlers) {
	pingTicker := time.NewTicker(pingPeriod)
	subscriber := a.getDBSubscriber()
	defer func() {
		subscriber.quitChan <- true
		pingTicker.Stop()
		ws.Close()
	}()

	for {
		select {
		case msg := <-subscriber.bufChan:
			if msg.MsgType == TenMinute {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteJSON(msg); err != nil {
					return
				}
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// writerFifteenSec runs in a goroutine for each connected WS client, but only
// emits a message on the connected socket if the monitor returns a new
// FifteenSecWind message.
func writerFifteenSec(ws *websocket.Conn, a *ApiHandlers) {
	pingTicker := time.NewTicker(pingPeriod)
	subscriber := a.getDBSubscriber()
	defer func() {
		subscriber.quitChan <- true
		pingTicker.Stop()
		ws.Close()
	}()

	for {
		select {
		case msg := <-subscriber.bufChan:
			if msg.MsgType == FifteenSecWind {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteJSON(msg); err != nil {
					return
				}
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
