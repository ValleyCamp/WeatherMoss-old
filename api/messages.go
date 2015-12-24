package api

import (
	//jww "github.com/spf13/jwalterweatherman"
	"time"
)

type MsgType string

const (
	FifteenSecWind MsgType = "FifteenSecWind"
	TenMinute              = "TenMinute"
)

type WSMessage struct {
	MsgType MsgType     `json:"msgType"`
	Payload interface{} `json:"payload"`
}

type FifteenSecWindMsg struct {
	ID            int       `json:"ID"`
	DateTime      time.Time `json:"DateTime"`
	WindDirCur    int       `json:"WindDirCur"`
	WindDirCurEng string    `json:"WindDirCurEng"`
	WindSpeedCur  float64   `json:"WindSpeedCur"`
}

type TenMinAllRow struct {
	ID              int
	DateTime        time.Time
	TempOutCur      float64
	HumOutCur       int
	PressCur        float64
	DewCur          float64
	HeatIdxCur      float64
	WindChillCur    float64
	TempInCur       float64
	HumInCur        int
	WindSpeedCur    float64
	WindAvgSpeedCur float64
	WindDirCur      int
	WindDirCurEng   string
	WindGust10      float64
	WindDirAvg10    int
	WindDirAvg10Eng string
	UVAvg10         float64
	UVMax10         float64
	SolarRadAvg10   float64
	SolarRadMax10   float64
	RainRateCur     float64
	RainDay         float64
	RainYest        float64
	RainMonth       float64
	RainYear        float64
}
