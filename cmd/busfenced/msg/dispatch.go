package msg

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
	Detect      string    `json:"detect"`      //侦察事件类型
	Meter       float64   `json:"meter"`       //半径
	TaskId      string    `json:"taskId"`      //任务ID，关联业务
	InvalidTime time.Time `json:"invalidTime"` //失效时间
}

func (g *Dispatch) String() string {
	return fmt.Sprintf("obuId=%s,lat=%f,lon=%f,meter=%f,detect=%s,taskId=%s,invalidtime=%s",
		g.Obuid, g.Lat, g.Lon, g.Meter, g.Detect, g.TaskId, g.InvalidTime.Format(time.RFC3339))
}

func (g *Dispatch) Json() string {
	bs, err := json.Marshal(g)
	if err != nil {
		return ""
	}

	value, _ := sjson.Set(string(bs), "invalidTime", g.InvalidTime.Format(time.RFC3339))

	return value
}
