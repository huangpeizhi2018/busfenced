package fenced

import (
	"fmt"
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
		arr, err := redis.Strings(conn.Do("BRPOP", s.cf.Source.GPSPoint, 0))
		if err != nil {
			return fmt.Errorf("fetchGPS BRPOP error, %s", err)
		}

		incr, err := redis.Int64(conn.Do("INCR", s.cf.Source.GPSTouch))
		if err != nil {
			s.log.Warn("fetchGPS INCR", zap.Error(err), zap.Int64("incr", incr), zap.String("touch", s.cf.Source.GPSTouch))
		}

		jstr := arr[1]
		lat := gjson.Get(jstr, "lat").Float()
		lon := gjson.Get(jstr, "lon").Float()

		//抛弃经/纬度可能出错的记录
		if !s.checkGPS(lat, lon, true) {
			s.log.Info("fetchGPS checkGPS failure", zap.String("gps", jstr))
			continue
		}

		m := GPS{
			Obuid:  gjson.Get(jstr, "obuid").String(),
			Lat:    lat,
			Lon:    lon,
			Height: gjson.Get(jstr, "height").Float(),
			Speed:  gjson.Get(jstr, "speed").Float(),
			Dir:    gjson.Get(jstr, "dir").Float(),
		}

		//gpstime日期分析
		gt := gjson.Get(jstr, "gpstime").String()
		t, err := time.ParseInLocation(time.RFC3339, gt, time.Local)
		if err != nil {
			s.log.Warn("fetchGPS parse gpstime error", zap.Error(err), zap.String("gps", jstr))
			continue
		}

		//抛弃超时或延迟消息
		if s.cf.GpsTimeOffset >= 0 {
			if math.Abs(time.Now().Sub(t).Seconds()) > s.cf.GpsTimeOffset {
				s.log.Info("fetchGPS gpstime delay",
					zap.Float64("gpstimeoffset", s.cf.GpsTimeOffset),
					zap.String("gps", jstr))
				continue
			}
		}

		m.GPSTime = t
		s.chanGPS <- &m //转入管道
	}
}

func (s *Server) updateGPS() error {
	enter := s.enter.Get()
	defer enter.Close()

	exit := s.exit.Get()
	defer exit.Close()

	for {
		for i := range s.chanGPS {
			if _, err := enter.Do("SET", s.cf.EnterFenced.Collection, i.Obuid, "POINT", i.Lat, i.Lon); err != nil {
				s.log.Info("updateGPS SET enterfenced error", zap.Error(err), zap.String("gps", i.Json()))
			}

			if _, err := exit.Do("SET", s.cf.ExitFenced.Collection, i.Obuid, "POINT", i.Lat, i.Lon); err != nil {
				s.log.Info("updateGPS SET exitfenced error", zap.Error(err), zap.String("gps", i.Json()))
			}
		}
	}

	return nil
}
