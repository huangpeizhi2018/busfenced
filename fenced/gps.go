package fenced

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tidwall/sjson"
)

//GPS消息
type GPS struct {
	Obuid     string    `json:"obuid"`     //GPS实体标识
	Lat       float64   `json:"lat"`       //纬度
	Lon       float64   `json:"lon"`       //经度
	Height    float64   `json:"height"`    //高度
	Dir       float64   `json:"dir"`       //方向
	Speed     float64   `json:"speed"`     //速度
	GPSTime   time.Time `json:"gpstime"`   //时间
	GPSUnix   int64     `json:"gpsunix"`   //GPSTIME转换的unixtime
	FetchUnix int64     `json:"fetchunix"` //分析获取消息时间转换的unixtime
}

func (g *GPS) String() string {
	return fmt.Sprintf("obuid=%s,lat=%f,lon=%f,height=%f,dir=%f,speed=%f,gpstime=%s,fetchtime=%s",
		g.Obuid, g.Lat, g.Lon, g.Height, g.Dir, g.Speed, g.GPSTime.Format(time.RFC3339), time.Unix(g.FetchUnix, 0).Format(time.RFC3339))
}

func (g *GPS) Json() string {
	bs, err := json.Marshal(g)
	if err != nil {
		return ""
	}

	jstr := string(bs)
	jstr, _ = sjson.Set(jstr, "gpstime", g.GPSTime.Format(time.RFC3339))

	return jstr
}
