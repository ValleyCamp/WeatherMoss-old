package api

import (
	"database/sql"
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

func (a *ApiHandlers) WsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			jww.FATAL.Println(err)
		}
		return
	}

	go writer(ws, a.db)
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
func writer(ws *websocket.Conn, db *sql.DB) {
	pingTicker := time.NewTicker(pingPeriod)
	dbTicker := time.NewTicker(dbPollPeriod)
	defer func() {
		pingTicker.Stop()
		dbTicker.Stop()
		ws.Close()
	}()

	// last-result tracking for this particularl connection. Initialize to epoch so that .After() returns true the first time no matter what
	var (
		lastFifteenSecRes = time.Unix(0, 0)
		last10MinuteRes   = time.Unix(0, 0)
	)

	for {
		select {
		case <-dbTicker.C:
			results, err := pollDB(db, &lastFifteenSecRes, &last10MinuteRes)

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

// pollDB is a helper function to writer() and is being called every dbTicker seconds for each websocket connection
func pollDB(db *sql.DB, lastFifteenSecRes *time.Time, last10MinuteRes *time.Time) ([]WSMessage, error) {
	res := make([]WSMessage, 0)

	rows, err := db.Query("SELECT * FROM housestation_15sec_wind ORDER BY ID DESC LIMIT 1")
	if err != nil {
		jww.ERROR.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		f := FifteenSecWindMsg{}
		err := rows.Scan(&f.ID, &f.DateTime, &f.WindDirCur, &f.WindDirCurEng, &f.WindSpeedCur)
		if err != nil {
			jww.ERROR.Println(err)
		}

		if f.DateTime.After(*lastFifteenSecRes) {
			r1 := WSMessage{
				MsgType: FifteenSecWind,
				Payload: f,
			}
			res = append(res, r1)
			*lastFifteenSecRes = f.DateTime
		}

	}
	err = rows.Err()
	if err != nil {
		jww.ERROR.Println(err)
	}

	// See if there's an updated 10 minute result
	trrows, err := db.Query("SELECT * FROM housestation_10min_all ORDER BY ID DESC LIMIT 1")
	if err != nil {
		jww.ERROR.Println(err)
	}
	defer trrows.Close()
	for trrows.Next() {
		t := TenMinAllRow{}
		err := trrows.Scan(&t.ID, &t.DateTime, &t.TempOutCur, &t.HumOutCur, &t.PressCur, &t.DewCur, &t.HeatIdxCur, &t.WindChillCur, &t.TempInCur,
			&t.HumInCur, &t.WindSpeedCur, &t.WindAvgSpeedCur, &t.WindDirCur, &t.WindDirCurEng, &t.WindGust10, &t.WindDirAvg10, &t.WindDirAvg10Eng,
			&t.UVAvg10, &t.UVMax10, &t.SolarRadAvg10, &t.SolarRadMax10, &t.RainRateCur, &t.RainDay, &t.RainYest, &t.RainMonth, &t.RainYear)
		if err != nil {
			jww.ERROR.Println(err)
		}

		if t.DateTime.After(*last10MinuteRes) {
			res = append(res, WSMessage{MsgType: TenMinute, Payload: t})
			*last10MinuteRes = t.DateTime
		}
	}

	if len(res) > 0 {
		return res, nil
	} else {
		return nil, nil
	}
}
