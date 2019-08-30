package fenced

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tidwall/sjson"
)

//GPS消息
type Dispatch struct {
	Obuid       string    `json:"obuId"`       //GPS实体标识
	Lat         float64   `json:"lat"`         //纬度
	Lon         float64   `json:"lon"`         //经度
	EnterMeter  float64   `json:"enterMeter"`  //进站半径
	ExitMeter   float64   `json:"exitMeter"`   //出站半径
	TaskId      string    `json:"taskId"`      //任务ID，关联业务
	InvalidTime time.Time `json:"invalidTime"` //失效时间
}

func (g *Dispatch) String() string {
	return fmt.Sprintf("obuId=%s,lat=%f,lon=%f,enterMeter=%f,exitMeter=%f,taskId=%s,invalidtime=%s",
		g.Obuid, g.Lat, g.Lon, g.EnterMeter, g.ExitMeter, g.TaskId, g.InvalidTime.Format(time.RFC3339))
}

func (g *Dispatch) Json() string {
	bs, err := json.Marshal(g)
	if err != nil {
		return ""
	}

	value, _ := sjson.Set(string(bs), "invalidTime", g.InvalidTime.Format(time.RFC3339))

	return value
}
