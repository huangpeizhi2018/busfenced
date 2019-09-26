package fenced

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ReneKroon/ttlcache"
	"github.com/gomodule/redigo/redis"
	"github.com/huangpeizhi2018/busfenced/fenced/version"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

type FenceType string

const (
	ENTER = FenceType("enter") //进站事件
	EXIT  = FenceType("exit")  //出站事件
)

type Server struct {
	cf *Conf

	sp    *redis.Pool //redis数据源
	tp    *redis.Pool //redis结果数据输出
	enter *redis.Pool //tile38
	exit  *redis.Pool //tile38

	chanGPS      chan *GPS
	chanDispatch chan *Dispatch

	enterCache *ttlcache.Cache
	exitCache  *ttlcache.Cache

	log   *zap.Logger
	start time.Time
}

//新建服务
func NewServer(c *Conf) (*Server, error) {
	s := &Server{
		cf:           c,
		chanGPS:      make(chan *GPS, c.ChanLen),
		chanDispatch: make(chan *Dispatch, c.ChanLen),
		start:        time.Now(),
	}

	var err error
	if err = s.setLog(); err != nil {
		return nil, err
	}

	if s.sp, err = s.setRedis(net.JoinHostPort(s.cf.Source.Addr, s.cf.Source.Port),
		s.cf.Source.Passwd,
		0,
		s.cf.Source.MaxIdel);
		err != nil {
		return nil, err
	}

	if s.tp, err = s.setRedis(net.JoinHostPort(s.cf.Target.Addr, s.cf.Target.Port),
		s.cf.Target.Passwd,
		0,
		s.cf.Target.MaxIdel);
		err != nil {
		return nil, err
	}

	if s.enter, err = s.setRedis(net.JoinHostPort(s.cf.EnterFenced.Addr, s.cf.EnterFenced.Port),
		"",
		0,
		0); err != nil {
		return nil, err
	}

	if s.exit, err = s.setRedis(net.JoinHostPort(s.cf.ExitFenced.Addr, s.cf.ExitFenced.Port),
		"",
		0,
		0); err != nil {
		return nil, err
	}

	//调度围栏过期时的回调处理。
	//1. 打印清除消息
	//2. DELHOOK
	//3. DEL 集合 obuid

	//进围栏服务
	enterExpirationCallback := func(key string, value interface{}) {
		s.log.Info("clean ENTER/fenced HOOKS  and obuid/lat/lon, dispatch expires",
			zap.String(key, value.(*Dispatch).Json()))

		obuid, _, valid := parseHook(key)
		if !valid {
			s.log.Warn("parseHook ENTER/fenced format incorrect",
				zap.String("hook", key))
			return
		}

		conn := s.enter.Get()
		defer conn.Close()

		if _, err := conn.Do("DELHOOK", key); err != nil {
			s.log.Warn("clean ENTER/fenced HOOKS, DELHOOK error",
				zap.Error(err))
		}

		if _, err := conn.Do("DEL", s.cf.EnterFenced.Collection, obuid); err != nil {
			s.log.Warn("DEL ENTER/fenced error",
				zap.String("collection", s.cf.EnterFenced.Collection),
				zap.String("id", obuid),
				zap.Error(err))
		}
	}

	//出围栏服务
	exitExpirationCallback := func(key string, value interface{}) {
		s.log.Info("clean EXIT/fenced HOOKS and obuid/lat/lon, dispatch expires, ",
			zap.String(key, value.(*Dispatch).Json()))

		obuid, _, valid := parseHook(key)
		if !valid {
			s.log.Warn("parseHook EXIT/fenced format incorrect",
				zap.String("hook", key))
			return
		}
		conn := s.exit.Get()
		defer conn.Close()

		if _, err := conn.Do("DELHOOK", key); err != nil {
			s.log.Warn("clean EXIT/fenced HOOKS, DELHOOK error",
				zap.Error(err))
		}

		if _, err := conn.Do("DEL", s.cf.ExitFenced.Collection, obuid); err != nil {
			s.log.Warn("DEL EXIT/fenced error",
				zap.String("collection", s.cf.ExitFenced.Collection),
				zap.String("id", obuid),
				zap.Error(err))
		}
	}

	//进围栏TTLCache初始化
	enterCache := ttlcache.NewCache()
	enterCache.SetExpirationCallback(enterExpirationCallback)
	s.enterCache = enterCache

	//出围栏TTLCache初始化
	exitCache := ttlcache.NewCache()
	exitCache.SetExpirationCallback(exitExpirationCallback)
	s.exitCache = exitCache

	return s, nil
}

//初始化日志
func (s *Server) setLog() error {
	//日志配置
	lvl := zap.NewAtomicLevel()
	err := lvl.UnmarshalText([]byte(s.cf.ZLog.Level))
	if err != nil {
		return err
	}

	cf := &zap.Config{
		Level:            lvl,
		Development:      s.cf.ZLog.Development,
		Encoding:         s.cf.ZLog.Encoding,
		OutputPaths:      s.cf.ZLog.OutputPaths,
		ErrorOutputPaths: s.cf.ZLog.ErrorOutputPaths,
	}
	cf.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	s.log, err = cf.Build()
	if err != nil {
		return err
	}
	defer s.log.Sync()

	return nil
}

//初始化redis
func (s *Server) setRedis(server string, password string, db int, maxidle int) (*redis.Pool, error) {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}

			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}

			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     maxidle,
		IdleTimeout: 240 * time.Second,
	}

	if _, err := pool.Get().Do("PING"); err != nil {
		return nil, err
	}

	return pool, nil
}

//启动围栏处理服务
func (s *Server) Run() error {
	defer s.Close()

	s.log.Info("busfenced startup", zap.String("version", version.String("busfenced")))

	errchan := make(chan error, 1)

	//进围栏事件存储
	go func() {
		s.log.Info("eventDump/ENTER startup")
		errchan <- s.eventDump(ENTER)
	}()

	//出围栏事件存储
	go func() {
		s.log.Info("eventDump/EXIT startup")
		errchan <- s.eventDump(EXIT)
	}()

	//转发Dispatch
	go func() {
		s.log.Info("fetchDispatch startup",
			zap.String("redis", net.JoinHostPort(s.cf.Target.Addr, s.cf.Target.Port)),
			zap.String("queue", s.cf.Source.DispatchPoint),
			zap.String("touch", s.cf.Source.DispatchTouch),
		)
		errchan <- s.fetchDispatch()
	}()

	//更新Dispatch信息到Hooks
	go func() {
		s.log.Info("updateDispatch startup")
		errchan <- s.updateDispatch()
	}()

	time.Sleep(5 * time.Second) //等待配置HOOKS

	//转发GPS
	go func() {
		s.log.Info("fetchGPS startup",
			zap.String("redis", net.JoinHostPort(s.cf.Source.Addr, s.cf.Source.Port)),
			zap.String("queue", s.cf.Source.GPSPoint),
			zap.String("touch", s.cf.Source.GPSTouch),
		)
		errchan <- s.fetchGPS()
	}()

	//更新GPS信息到围栏处理集合
	go func() {
		s.log.Info("updateGPS startup")
		errchan <- s.updateGPS()
	}()

	//运行状态接口
	if s.cf.Stats.Valid {
		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Please visit /debug/vars")
			})

			s.log.Info("stats startup",
				zap.String("addr", net.JoinHostPort(s.cf.Stats.Addr, s.cf.Stats.Port)),
			)

			errchan <- http.ListenAndServe(net.JoinHostPort(s.cf.Stats.Addr, s.cf.Stats.Port), nil)
		}()
	}

	//TILE38 AOFSHRINK
	if s.cf.AOFShrink.Valid {
		go func() {
			for {
				s.log.Info("aofshrink startup",
					zap.Time("starttime", time.Now()),
					zap.Int64("interval.seconds", s.cf.AOFShrink.Seconds))

				func() {
					enterConn := s.enter.Get()
					defer enterConn.Close()

					res, err := redis.String(enterConn.Do("AOFSHRINK"))
					if err != nil {
						s.log.Warn("enter aofshrink", zap.Error(err))
					}

					exitConn := s.enter.Get()
					defer exitConn.Close()

					res, err = redis.String(enterConn.Do("AOFSHRINK"))
					if err != nil {
						s.log.Warn("exit aofshrink", zap.Error(err))
					}

					s.log.Info("aofshrink finished", zap.String("res", res))
				}()

				s.log.Info("aofshrink end",
					zap.String("endtime", time.Now().Format(time.RFC3339Nano)),
					zap.Int64("interval.seconds", s.cf.AOFShrink.Seconds))

				time.Sleep(time.Second * time.Duration(s.cf.AOFShrink.Seconds))
			}
		}()
	}

	//结束
	return <-errchan
}

//Redis发布点结构
type RedisPubpoint struct {
	Host    string
	Port    string
	Channel string
}

//围栏事件转储
func (s *Server) eventDump(ft FenceType) error {
	pubpoint := new(RedisPubpoint)

	var err error
	switch ft {
	case ENTER:
		err = pubpoint.parsePubpoint(s.cf.EnterFenced.PubPoint)
	case EXIT:
		err = pubpoint.parsePubpoint(s.cf.ExitFenced.PubPoint)
	default:
		panic("eventDump FenceType error")
	}

	if err != nil {
		return fmt.Errorf("eventDump/%s pubpoint parsePubpoint error, %s", ft, err.Error())
	}

	//发布点与输出redis不相同
	if !(s.cf.Target.Addr == pubpoint.Host && s.cf.Target.Port == pubpoint.Port) {
		return fmt.Errorf("eventDump/%s pubpoint is inconsistent with the target's hostname or port", ft)
	}

	conn := s.tp.Get()
	defer conn.Close()

	pub := redis.PubSubConn{Conn: conn}
	if err := pub.Subscribe(pubpoint.Channel); err != nil {
		return err
	}

	for {
		switch v := pub.Receive().(type) {
		case redis.Message:
			jstr := string(v.Data)

			err := func(jstr string, ft FenceType) error {
				tc := s.tp.Get()
				defer tc.Close()

				//丢弃，不属于该车辆的触发事件。
				hook := gjson.Get(jstr, "hook").String()
				id := gjson.Get(jstr, "id").String()
				if !strings.HasPrefix(hook, id) {
					s.log.Debug("eventDump event not match",
						zap.String("id", id),
						zap.String("hook", hook),
						zap.String("FenceType", string(ft)))
					return nil
				}

				//丢弃，hook格式不正确的触发事件。
				_, taskid, valid := parseHook(hook)
				if !valid {
					s.log.Warn("eventDump event hook format incorrect",
						zap.String("hook", hook),
						zap.String("FenceType", string(ft)))
					return nil
				}

				//补充，关联业务ID信息。
				var err error
				jstr, err = sjson.Set(jstr, "task.id", taskid)
				if err != nil {
					s.log.Warn("eventDump json set",
						zap.String("task.id", taskid),
						zap.String("jstr", jstr),
						zap.String("FenceType", string(ft)))
					return nil
				}

				//属于，进站事件围栏服务触发的事件
				if ft == ENTER {
					s.log.Info("EventDump ENTER/fenced event success",
						zap.String("jstr", jstr),
						zap.String("FenceType", string(ft)))

					_, err := tc.Do("LPUSH", s.cf.Target.EnterPoint, jstr)
					if err != nil {
						return fmt.Errorf("EventDump/%s LPUSH error, %s", ft, err)
					}

					if _, err := tc.Do("INCR", s.cf.Target.EnterTouch); err != nil {
						s.log.Warn("EventDump INCR",
							zap.String("FenceType", string(ft)), zap.Error(err))
					}

					//即时清理HOOK定义
					if s.cf.EnterFenced.DeleteNow {
						//因为FENCED都开启了enter/exit事件，所以只有进站事件，才删除HOOK定义。
						if gjson.Get(jstr, "detect").String() == string(ENTER) {
							func() {
								conn := s.enter.Get()
								defer conn.Close()

								_, _ = conn.Do("DELHOOK", hook)
							}()
						}
					}
				}

				//属于，出站事件围栏服务触发的事件
				if ft == EXIT {
					distance := gjson.Get(jstr, "distance").Int()
					//怀疑围栏触发的GPS有问题
					if distance > s.cf.ExitFenced.Distance {
						s.log.Info("EventDump trigger GPS coordinates are incorrect ",
							zap.String("FenceType", string(ft)),
							zap.String("jstr", jstr))
					} else {
						s.log.Info("EventDump EXIT/fenced event success",
							zap.String("jstr", jstr),
							zap.String("FenceType", string(ft)))

						_, err := tc.Do("LPUSH", s.cf.Target.ExitPoint, jstr)
						if err != nil {
							return fmt.Errorf("EventDump/%s LPUSH error, %s", ft, err)
						}

						if _, err := tc.Do("INCR", s.cf.Target.ExitTouch); err != nil {
							s.log.Warn("EventDump INCR",
								zap.String("FenceType", string(ft)),
								zap.Error(err))
						}

						//即时清理HOOK定义
						if s.cf.ExitFenced.DeleteNow {
							//因为FENCED都开启了enter/exit事件，所以事件与围栏相同时，才删除HOOK。
							if gjson.Get(jstr, "detect").String() == string(EXIT) {
								func() {
									conn := s.exit.Get()
									defer conn.Close()

									_, _ = conn.Do("DELHOOK", hook)
								}()
							}
						}
					}
				}

				return nil
			}(jstr, ft)

			if err != nil {
				s.log.Warn("eventDump",
					zap.String("jstr", jstr),
					zap.Error(err))
				return err
			}
		case redis.Subscription:
			s.log.Info("eventDump",
				zap.String("FenceType", string(ft)),
				zap.String("channel", v.Channel),
				zap.String("kind", v.Kind),
				zap.Int("count", v.Count),
			)
		case error:
			return fmt.Errorf("%s eventDump error, %s", string(ft), v)
		}
	}

	return nil
}

//分析REDIS事件发布点的URL，形成Endpoint结构。
//Redis URL示例 redis://127.0.0.1:6390/pub-enterfenced
func (endpoint *RedisPubpoint) parsePubpoint(s string) error {
	rawUrl := s
	if !strings.HasPrefix(s, "redis:") {
		return fmt.Errorf("endpoint protocol error, url [%s]", rawUrl)
	}

	s = s[strings.Index(s, ":")+1:]
	if !strings.HasPrefix(s, "//") {
		return fmt.Errorf("missing the two slashes, url [%s]", rawUrl)
	}

	sqp := strings.Split(s[2:], "?")
	sp := strings.Split(sqp[0], "/")
	s = sp[0]
	if s == "" {
		return fmt.Errorf("missing host, url [%s]", rawUrl)
	}

	dp := strings.Split(s, ":")
	endpoint.Host = dp[0]
	_, err := strconv.ParseUint(dp[1], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid redis url port, url [%s]")
	}
	endpoint.Port = dp[1]

	if len(sp) > 1 {
		var err error
		endpoint.Channel, err = url.QueryUnescape(sp[1])
		if err != nil {
			return fmt.Errorf("invalid redis channel name, [%s]", rawUrl)
		}
	} else {
		return fmt.Errorf("missing redis channel name, [%s]", rawUrl)
	}

	return nil
}

func (s *Server) Close() error {
	errmsg := []string{}

	if err := s.log.Sync(); err != nil {
		errmsg = append(errmsg, err.Error())
	}

	if err := s.sp.Close(); err != nil {
		errmsg = append(errmsg, err.Error())
	}

	if err := s.tp.Close(); err != nil {
		errmsg = append(errmsg, err.Error())
	}

	if err := s.enter.Close(); err != nil {
		errmsg = append(errmsg, err.Error())
	}

	if err := s.exit.Close(); err != nil {
		errmsg = append(errmsg, err.Error())
	}

	s.enterCache.Purge()

	if len(errmsg) > 0 {
		return fmt.Errorf(strings.Join(errmsg, "\n"))
	}

	return nil
}
