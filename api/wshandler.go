package api

import (
	//"database/sql"
	_ "github.com/go-sql-driver/mysql"
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

	// How often do we check the DB for an update?
	dbPollPeriod = 1 * time.Second
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type WSMessage struct {
	MsgType string `json:"msgType"`
	Payload string `json:"payload"`
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			jww.FATAL.Println(err)
		}
		return
	}

	go writer(ws)
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

// writer runs in a goroutine for each connected WS client.
func writer(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	dbTicker := time.NewTicker(dbPollPeriod)
	defer func() {
		pingTicker.Stop()
		dbTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-dbTicker.C:
			results, err := pollDB()
			if err != nil {
				// Errors have been logged upstream in polldb.
				// TODO: Can we return an error state to the client? Do they need to retry?
				return
			}

			for _, res := range results {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteJSON(res); err != nil {
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

func pollDB() ([]WSMessage, error) {
	// TODO: Poll db for updates in the last dbPollPeriod, return a list of all changes.

	return nil, nil
}
