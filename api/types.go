package api

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
	"time"
)

const (
	// How often does the monitor check the DB for an update?
	dbPollPeriod = 1 * time.Second
)

type ApiHandlers struct {
	db      *sql.DB
	monitor *dbMonitor
}

func NewApiHandlers(d *sql.DB) *ApiHandlers {
	a := &ApiHandlers{
		db: d,
		monitor: &dbMonitor{
			lastFifteenSecResTime: time.Unix(0, 0),
			lastTenMinResTime:     time.Unix(0, 0),
			subscribers:           make(([]*subscriber), 0),
		},
	}
	go a.runMonitor()

	return a
}

type subscriber struct {
	bufChan  chan WSMessage
	quitChan chan bool

	sync.RWMutex
}

func getSubscriber() *subscriber {
	return &subscriber{
		bufChan:  make(chan WSMessage, 110),
		quitChan: make(chan bool),
	} // TODO: Tune arbitrary size 110 as necessary. Designed to allow full backfill of 100 results to a combined subscriber
}

type dbMonitor struct {
	lastFifteenSecResTime time.Time
	lastTenMinResTime     time.Time
	latestFifteenSecRes   WSMessage
	latestTenMinRes       WSMessage
	subscribers           []*subscriber
	sync.RWMutex
}

// getDBObserver gets a new channel that can be used to listen for database updates
func (a *ApiHandlers) getDBSubscriber() *subscriber {
	ns := getSubscriber()
	a.monitor.Lock()
	a.monitor.subscribers = append(a.monitor.subscribers, ns)
	cl := len(a.monitor.subscribers)
	a.monitor.Unlock()
	jww.INFO.Println("Subscriber Created. There are now", cl, "subscribers.")
	//defer a.notifyOfLatest(ns)
	defer a.backfill(ns)
	return ns
}

// notifyOfLatest adds the most recent values the monitor has collected to a (usually newly-created)
// observer's channel, without blocking and preventing the observer from picking them up.
func (a *ApiHandlers) notifyOfLatest(s *subscriber) {
	go func(ia *ApiHandlers, is *subscriber) {
		is.bufChan <- ia.monitor.latestFifteenSecRes
		is.bufChan <- ia.monitor.latestTenMinRes
	}(a, s)
}

// backfill gets the last 40 results of each table and pushes them, in-order, onto the websocket queue.
// This allows the new subscriber to recieve enough data to fill in charts and graphs immediately.
func (a *ApiHandlers) backfill(s *subscriber) {
	go func(ia *ApiHandlers, is *subscriber) {
		rows, err := a.db.Query("SELECT * FROM ( SELECT * FROM housestation_15sec_wind ORDER BY ID DESC LIMIT 50 ) AS t ORDER BY ID")
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

			r1 := WSMessage{
				MsgType: FifteenSecWind,
				Payload: f,
			}

			is.bufChan <- r1

		}
		err = rows.Err()
		if err != nil {
			jww.ERROR.Println(err)
		}

		// See if there's an updated 10 minute result
		trrows, err := a.db.Query("SELECT * FROM (SELECT * FROM housestation_10min_all ORDER BY ID DESC LIMIT 50) AS t ORDER BY ID")
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

			r2 := WSMessage{MsgType: TenMinute, Payload: t}
			is.bufChan <- r2
		}
	}(a, s)
}

// runMonitor starts the monitor for this ApiHandlers objects
func (a *ApiHandlers) runMonitor() {
	dbTicker := time.NewTicker(dbPollPeriod)
	cleanupTicker := time.NewTicker(5 * time.Second)
	defer func() {
		dbTicker.Stop()
		cleanupTicker.Stop()
	}()

	for {
		select {
		case <-cleanupTicker.C:
			a.monitor.RLock()
			for i := len(a.monitor.subscribers) - 1; i >= 0; i-- {
				s := a.monitor.subscribers[i]
				select {
				case <-s.quitChan:
					// This subscriber has been set to quit, so we'll remove it from our loop
					a.monitor.subscribers = append(a.monitor.subscribers[:i], a.monitor.subscribers[i+1:]...)
					sc := len(a.monitor.subscribers)
					close(s.bufChan) // TODO: Is this possible to panic? Any chance we async read after the close?
					jww.INFO.Println("Subscriber Closed. There are now", sc, "subscribers.")
				default:
					// Nothing, we just carry on
				}
			}
			a.monitor.RUnlock()
		case <-dbTicker.C:
			results, err := pollDB(a)

			if err != nil {
				// Errors have been logged upstream in polldb.
				// TODO: Can we return an error state to the client? Do they need to retry?
				return
			}

			// Notify all observers of update
			for i := len(a.monitor.subscribers) - 1; i >= 0; i-- {
				s := a.monitor.subscribers[i]

				for _, r := range results {
					select {
					case s.bufChan <- r:
						// Nothing to do, we've written out to the subscriber
					default:
						// This should only occur if we coludn't write to the buffered channel, which only happens if it's full
						// TODO: could this also occur when the pollDB doesn't have a result?
						jww.ERROR.Println("subscriber buffer overflowed!")
					}
				}
			}
		}
	}
}

// pollDB is a helper function to runMonitor() and is being called every dbTicker seconds
// pollDB only outputs results in the case that the latest information from the database
// is newer than the most recent data cached in the monitor.
func pollDB(a *ApiHandlers) ([]WSMessage, error) {
	res := make([]WSMessage, 0)

	rows, err := a.db.Query("SELECT * FROM housestation_15sec_wind ORDER BY ID DESC LIMIT 1")
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

		if f.DateTime.After(a.monitor.lastFifteenSecResTime) {
			r1 := WSMessage{
				MsgType: FifteenSecWind,
				Payload: f,
			}
			res = append(res, r1)
			a.monitor.lastFifteenSecResTime = f.DateTime
			a.monitor.latestFifteenSecRes = r1
		}

	}
	err = rows.Err()
	if err != nil {
		jww.ERROR.Println(err)
	}

	// See if there's an updated 10 minute result
	trrows, err := a.db.Query("SELECT * FROM housestation_10min_all ORDER BY ID DESC LIMIT 1")
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

		if t.DateTime.After(a.monitor.lastTenMinResTime) {
			r2 := WSMessage{MsgType: TenMinute, Payload: t}
			res = append(res, r2)
			a.monitor.lastTenMinResTime = t.DateTime
			a.monitor.latestTenMinRes = r2
		}
	}

	if len(res) > 0 {
		return res, nil
	} else {
		return nil, nil
	}
}
