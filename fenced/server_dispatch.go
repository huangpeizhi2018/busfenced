package fenced

import (
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

//从数据源中提取dispatch信息过滤后放于内存管道
func (s *Server) fetchDispatch() error {
	conn := s.sp.Get()
	defer conn.Close()

	for {
		bp, err := redis.Strings(conn.Do("BRPOP", s.cf.Source.DispatchPoint, 0))
		if err != nil {
			s.log.Warn("fetchDispatch BRPOP", zap.String("queue", s.cf.Source.DispatchPoint), zap.Error(err))
			//5秒后重试
			time.Sleep(5 * time.Second)
			continue
		}

		incr, err := redis.Int64(conn.Do("INCR", s.cf.Source.DispatchTouch))
		if err != nil {
			s.log.Warn("fetchDispatch INCR", zap.Int64("incr", incr), zap.String("touch", s.cf.Source.DispatchTouch), zap.Error(err))
		}

		jstr := bp[1]
		lat := gjson.Get(jstr, "lat").Float()
		lon := gjson.Get(jstr, "lon").Float()

		//抛弃经/纬度可能出错的记录
		if !s.checkGPS(lat, lon, true) {
			s.log.Info("fetchDispatch checkGPS failure, discard this dispatch message", zap.String("dispatch", jstr))
			continue
		}

		enter := gjson.Get(jstr, "enterMeter").Float()
		exit := gjson.Get(jstr, "exitMeter").Float()
		//检查米数是否符合要求
		if !s.checkMeter(enter, exit) {
			s.log.Info("fetchDispatch checkMeter failure, discard this dispatch message", zap.String("dispatch", jstr))
			continue
		}

		m := Dispatch{
			Obuid:      gjson.Get(jstr, "obuId").String(),
			Lat:        lat,
			Lon:        lon,
			EnterMeter: enter,
			ExitMeter:  exit,
			TaskId:     gjson.Get(jstr, "taskId").String(),
		}

		//invalidTime日期分析
		it := gjson.Get(jstr, "invalidTime").String()
		t, err := time.ParseInLocation(time.RFC3339, it, time.Local)
		if err != nil {
			s.log.Warn("fetchDispatch parse invalidTime failure, discard this dispatch message", zap.Error(err), zap.String("dispatch", jstr))
			continue
		}

		//抛弃超时或延迟消息
		if t.Before(time.Now()) {
			s.log.Info("fetchDispatch invalidTime before now, discard this dispatch message", zap.String("dispatch", jstr))
			continue
		}

		m.InvalidTime = t
		s.chanDispatch <- &m //转入管道
	}
}

func (s *Server) updateDispatch() error {
	enter := s.enter.Get()
	defer enter.Close()

	exit := s.exit.Get()
	defer exit.Close()

	for {
		for i := range s.chanDispatch {
			lat := strconv.FormatFloat(i.Lat, 'f', -1, 64)
			lon := strconv.FormatFloat(i.Lon, 'f', -1, 64)
			enterM := strconv.FormatFloat(i.EnterMeter, 'f', -1, 64)
			exitM := strconv.FormatFloat(i.ExitMeter, 'f', -1, 64)

			//命令字符串
			enterHook := strings.Join(
				[]string{"SETHOOK",
					i.Obuid + ":" + i.TaskId,
					s.cf.EnterFenced.PubPoint,
					"NEARBY", s.cf.EnterFenced.Collection, "FENCE", "DETECT", "enter,exit", "COMMANDS", "set", "POINT", lat, lon, enterM}, " ")
			exitHook := strings.Join(
				[]string{"SETHOOK",
					i.Obuid + ":" + i.TaskId,
					s.cf.ExitFenced.PubPoint,
					"NEARBY", s.cf.EnterFenced.Collection, "FENCE", "DETECT", "enter,exit", "COMMANDS", "set", "POINT", lat, lon, exitM}, " ")

			key := i.Obuid + ":" + i.TaskId
			//进围栏事件触发
			s.log.Info("updateDispatch SETHOOK enter fenced", zap.String("hook", enterHook))
			if _, err := enter.Do("SETHOOK",
				key,
				s.cf.EnterFenced.PubPoint,
				"NEARBY", s.cf.EnterFenced.Collection, "FENCE", "DETECT", "enter,exit", "COMMANDS", "set", "POINT", i.Lat, i.Lon, i.EnterMeter); err != nil {
				s.log.Warn("updateDispatch SETHOOK enter error", zap.Error(err), zap.String("hook", enterHook), zap.String("dispatch", i.Json()))
				return err
			}
			s.enterCache.SetWithTTL(key, i, i.InvalidTime.Sub(time.Now()))

			//出围栏事件触发
			s.log.Info("updateDispatch SETHOOK exit fenced", zap.String("hook", exitHook))
			if _, err := exit.Do("SETHOOK",
				key,
				s.cf.ExitFenced.PubPoint,
				"NEARBY", s.cf.ExitFenced.Collection, "FENCE", "DETECT", "enter,exit", "COMMANDS", "set", "POINT", i.Lat, i.Lon, i.ExitMeter); err != nil {
				s.log.Warn("updateDispatch SETHOOK exit error", zap.Error(err), zap.String("hook", enterHook), zap.String("dispatch", i.Json()))
				return err
			}
			s.exitCache.SetWithTTL(key, i, i.InvalidTime.Sub(time.Now()))
		}
	}

	return nil
}
