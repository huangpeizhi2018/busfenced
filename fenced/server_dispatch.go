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
			s.log.Warn("fetchDispatch BRPOP",
				zap.String("queue", s.cf.Source.DispatchPoint),
				zap.Error(err))
			//5秒后重试
			time.Sleep(5 * time.Second)
			continue
		}

		incr, err := redis.Int64(conn.Do("INCR", s.cf.Source.DispatchTouch))
		if err != nil {
			s.log.Warn("fetchDispatch INCR", zap.Int64("incr", incr),
				zap.String("touch", s.cf.Source.DispatchTouch),
				zap.Error(err))
		}

		jstr := bp[1]
		lat := gjson.Get(jstr, "lat").Float()
		lon := gjson.Get(jstr, "lon").Float()

		//抛弃经/纬度可能出错的记录
		if !s.checkGPS(lat, lon) {
			s.log.Info("fetchDispatch checkGPS failure, discard this dispatch message",
				zap.String("dispatch", jstr))
			continue
		}

		meter := gjson.Get(jstr, "meter").Float()
		//检查米数是否符合要求
		if !s.checkMeter(meter) {
			s.log.Info("fetchDispatch checkMeter failure, discard this dispatch message",
				zap.String("dispatch", jstr))
			continue
		}

		detect := gjson.Get(jstr, "detect").String()
		if !(detect == string(ENTER) || detect == string(EXIT)) {
			s.log.Info("fetchDispatch detect error, discard this dispatch message",
				zap.String("dispatch", jstr))
			continue
		}

		//invalidTime日期分析
		it := gjson.Get(jstr, "invalidTime").String()
		t, err := time.ParseInLocation(time.RFC3339, it, time.Local)
		if err != nil {
			s.log.Warn("fetchDispatch parse invalidTime failure, discard this dispatch message",
				zap.Error(err),
				zap.String("dispatch", jstr))
			continue
		}

		//抛弃超时或延迟消息
		if t.Before(time.Now()) {
			s.log.Info("fetchDispatch invalidTime before now, discard this dispatch message",
				zap.String("dispatch", jstr))
			continue
		}

		m := Dispatch{
			Obuid:       gjson.Get(jstr, "obuId").String(),
			Lat:         lat,
			Lon:         lon,
			Meter:       meter,
			Detect:      detect,
			InvalidTime: t,
			TaskId:      gjson.Get(jstr, "taskId").String(),
		}
		s.chanDispatch <- &m //转入管道
	}
}

//更新HOOK
func (s *Server) updateDispatch() error {
	enter := s.enter.Get()
	defer enter.Close()

	exit := s.exit.Get()
	defer exit.Close()

	for i := range s.chanDispatch {
		lat := strconv.FormatFloat(i.Lat, 'f', -1, 64)
		lon := strconv.FormatFloat(i.Lon, 'f', -1, 64)
		meter := strconv.FormatFloat(i.Meter, 'f', -1, 64)

		//命令字符串
		var hook string
		key := i.Obuid + ":" + i.TaskId
		if i.Detect == string(ENTER) {
			hook = strings.Join(
				[]string{"SETHOOK",
					key,
					s.cf.EnterFenced.PubPoint,
					"NEARBY",
					s.cf.EnterFenced.Collection,
					"DISTANCE", "FENCE", "DETECT", "enter", "COMMANDS", "set", "POINT", lat, lon, meter}, " ")
		} else if i.Detect == string(EXIT) {
			hook = strings.Join(
				[]string{"SETHOOK",
					key,
					s.cf.ExitFenced.PubPoint,
					"NEARBY",
					s.cf.EnterFenced.Collection,
					"DISTANCE", "FENCE", "DETECT", "exit", "COMMANDS", "set", "POINT", lat, lon, meter}, " ")
		} else {
			panic("never come here!")
		}

		s.log.Info("updateDispatch SETHOOK",
			zap.String("detect", i.Detect),
			zap.String("hook", hook))

		//执行如下操作：
		//- 设置HOOK。
		//- 清除OBU历史GPS记录。
		//- 设置过期时间触发。
		if i.Detect == string(ENTER) {
			if _, err := enter.Do("SETHOOK",
				key,
				s.cf.EnterFenced.PubPoint,
				"NEARBY",
				s.cf.EnterFenced.Collection,
				"DISTANCE", "FENCE", "DETECT", "enter", "COMMANDS", "set", "POINT", i.Lat, i.Lon, i.Meter);
				err != nil {
				s.log.Warn("updateDispatch SETHOOK ENTER error",
					zap.Error(err), zap.String("hook", hook),
					zap.String("dispatch", i.Json()))
				return err
			}

			if _, err := enter.Do("DEL", s.cf.EnterFenced.Collection, i.Obuid); err != nil {
				s.log.Warn("updateDispatch DEL ENTER error",
					zap.Error(err),
					zap.String("collection", s.cf.EnterFenced.Collection),
					zap.String("obuid", i.Obuid),
					zap.String("dispatch", i.Json()))
				return err
			}

			s.enterCache.SetWithTTL(key, i, i.InvalidTime.Sub(time.Now()))
		} else if i.Detect == string(EXIT) {
			if _, err := exit.Do("SETHOOK",
				key,
				s.cf.ExitFenced.PubPoint,
				"NEARBY",
				s.cf.ExitFenced.Collection,
				"DISTANCE", "FENCE", "DETECT", "exit", "COMMANDS", "set", "POINT", i.Lat, i.Lon, i.Meter);
				err != nil {
				s.log.Warn("updateDispatch SETHOOK EXIT error",
					zap.Error(err), zap.String("hook", hook),
					zap.String("dispatch", i.Json()))
				return err
			}

			if _, err := enter.Do("DEL", s.cf.ExitFenced.Collection, i.Obuid); err != nil {
				s.log.Warn("updateDispatch DEL EXIT error",
					zap.Error(err),
					zap.String("collection", s.cf.ExitFenced.Collection),
					zap.String("obuid", i.Obuid),
					zap.String("dispatch", i.Json()))
				return err
			}

			s.exitCache.SetWithTTL(key, i, i.InvalidTime.Sub(time.Now()))
		} else {
			panic("never come here!")
		}
	}

	return nil
}
