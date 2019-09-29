package fenced

import (
	"math"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

//从数据源中提取GPS信息过滤后放于内存管道
func (s *Server) fetchGPS() error {
	conn := s.sp.Get()
	defer conn.Close()

	for {
		bp, err := redis.Strings(conn.Do("BRPOP", s.cf.Source.GPSPoint, 0))
		if err != nil {
			s.log.Warn("fetchGPS BRPOP",
				zap.String("queue", s.cf.Source.GPSPoint),
				zap.Error(err))
			//5秒后重试
			time.Sleep(5 * time.Second)
			continue
		}

		incr, err := redis.Int64(conn.Do("INCR", s.cf.Source.GPSTouch))
		if err != nil {
			s.log.Warn("fetchGPS INCR",
				zap.Int64("incr", incr),
				zap.String("touch", s.cf.Source.GPSTouch),
				zap.Error(err))
		}

		jstr := bp[1]
		lat := gjson.Get(jstr, "lat").Float()
		lon := gjson.Get(jstr, "lon").Float()

		//抛弃经/纬度出错的记录
		if !s.checkGPS(lat, lon) {
			s.log.Info("fetchGPS checkGPS failure, discard this GPS message",
				zap.String("gps", jstr))
			continue
		}

		m := GPS{
			Obuid:     gjson.Get(jstr, "obuid").String(),
			Lat:       lat,
			Lon:       lon,
			Height:    gjson.Get(jstr, "height").Float(),
			Speed:     gjson.Get(jstr, "speed").Float(),
			Dir:       gjson.Get(jstr, "dir").Float(),
			FetchUnix: time.Now().Unix(),
		}

		//gpstime日期分析
		gt := gjson.Get(jstr, "gpstime").String()
		t, err := time.ParseInLocation(time.RFC3339, gt, time.Local)
		if err != nil {
			s.log.Warn("fetchGPS parse gpstime failure, discard this GPS message",
				zap.Error(err),
				zap.String("gps", jstr))
			continue
		}

		//抛弃超时或延迟消息
		if s.cf.GpsTimeOffset >= 0 {
			if math.Abs(time.Now().Sub(t).Seconds()) > s.cf.GpsTimeOffset {
				s.log.Debug("fetchGPS gpstime delay over the threshold, discard this GPS message",
					zap.Float64("threshold", s.cf.GpsTimeOffset),
					zap.String("gps", jstr))
				continue
			}
		}

		m.GPSTime = t
		m.GPSUnix = t.Unix()
		s.chanGPS <- &m //转入管道
	}
}

//更新围栏关联GPS集合
func (s *Server) updateGPS() error {
	enter := s.enter.Get()
	defer enter.Close()

	exit := s.exit.Get()
	defer exit.Close()

	for i := range s.chanGPS {
		//追加geo属性
		bs, err := s.mkGeojson("POINT", *i)
		if err != nil {
			s.log.Warn("updateGPS mkGeojson error", zap.Error(err),
				zap.String("gps", i.Json()))
			continue
		}

		if _, err := enter.Do("SET", s.cf.EnterFenced.Collection, i.Obuid,
			"OBJECT", string(bs)); err != nil {
			s.log.Warn("updateGPS SET enter fenced error",
				zap.Error(err),
				zap.String("gps", i.Json()))
			return err
		}

		if _, err := exit.Do("SET", s.cf.ExitFenced.Collection, i.Obuid,
			"OBJECT", string(bs)); err != nil {
			s.log.Warn("updateGPS SET exit fenced error",
				zap.Error(err),
				zap.String("gps", i.Json()))
			return err
		}
	}

	return nil
}
