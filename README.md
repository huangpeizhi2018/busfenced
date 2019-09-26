# busfenced/公交总站围栏处理

## 围栏服务需注意之处
- 围栏事件：有进才有出。没有“进”围栏事件，就不会产生“出”围栏事件。
- 围栏定义包含“到期时间”，当“到期时间”达到时，会清理围栏定义及对应终端前一条GPS信息。
- 围栏定义配置中的deletenow参数为true时，则触发事件后，即刻会删除围栏定义。需补充对应终端GPS信息的清理。

### 非法GPS干扰的问题
- 非法GPS造成的一次出围栏事件，系统会根据出围栏触发消息中的distance属性值进行判断，如果距离值大于“配置文件中规定阀值”，则认为此事件不正确，会有日志告警，不会执行DELHOOK动作。
- 但会认为下一个或下某个GPS正常点，会产生进围栏事件！
- 存在问题：如果下某个GPS正常点不产生进围栏事件，则会造成“丢失正常的出围栏事件”。
- 对GPS正确性只有简单判断。
```
	if lat < 20 || lon < 110 || lat > 30 || lon > 120 {
		ret = false
		s.log.Debug("lat/lon invalid", zap.Float64("lat", lat), zap.Float64("lon", lon))
	}
```

## 围栏分析架构
<p align="center" style="text-align:center;">
  <img src="https://github.com/huangpeizhi2018/busfenced/blob/master/docs/fenced.jpg" width="500" />
</p>

## 运行

### 进程管理
- 使用supervisor管理busfenced服务相关进程。
```
/opt/supervisor/bin/supervisord -c /opt/supervisor/conf/supervisord.conf -d
```
- [supervisord](https://github.com/ochinchina/supervisord)
- 监控界面
<p align="center" style="text-align:center;">
  <img src="https://github.com/huangpeizhi2018/busfenced/blob/master/docs/go-supervisor.png" width="500" />
</p>

### 目录结构
```
/opt/busfenced
├── bin
│   ├── busfenced
│   ├── tile38-benchmark
│   ├── tile38-cli
│   └── tile38-server
├── conf
│   └── busfenced.yaml
├── log
│   └── busfenced.log
├── pid
│   ├── busfenced.pid
│   ├── tile38_7875.pid
│   └── tile38_7876.pid
└── tile38
    ├── enterfence
    │   ├── config
    │   └── queue.db
    └── exitfence
        ├── config
        └── queue.db
```

## 测试
### GPS源信息格式
```ruby
{
    "speed": 0, 
    "gpstime": "2019-09-04T08:55:35+08:00", 
    "lon": 113.3263889, 
    "lat": 23.1191667, 
    "dir": 274, 
    "obuid": "989854"
}
```

### 调度源信息格式
```ruby
{
    "obuId": "941184", 
    "lat": 23.1152778, 
    "lon": 113.2825,
    "detect": "enter"或"exit" 
    "meter": 100, 
    "taskId": "8", 
    "invalidTime": "2019-09-03T16:55:00+08:00"
}
```

### ruby/redis操作示例
```ruby
require 'redis'
r = Redis.new(:host=>'10.88.100.132', :port=>6390)
# 围栏指令
# SETHOOK 933526:1568048402031 redis://127.0.0.1:6390/pub-enterfenced NEARBY busgps FENCE DETECT enter,exit COMMANDS set POINT 23.145199 113.35396 200"}
# SETHOOK 933526:1568048402031 redis://127.0.0.1:6390/pub-exitfenced NEARBY busgps FENCE DETECT enter,exit COMMANDS set POINT 23.145199 113.35396 200"}   
dispatch = '{"obuId": "123456","lat": 23.145199,"lon": 113.35396,"meter": 200,"taskId": "1234567890","detect": "enter","invalidTime": "2019-09-11T17:30:00+08:00"}'
r.lpush('queue.bus.dispatch',dispatch)
dispatch = '{"obuId": "123456","lat": 23.145199,"lon": 113.35396,"meter": 200,"taskId": "1234567890","detect": "exit","invalidTime": "2019-09-11T17:30:00+08:00"}'
r.lpush('queue.bus.dispatch',dispatch)
# GPS信息 
## 进入围栏
gps = '{ "obuid": "123456", "lat": 23.1463889, "lon": 113.3525000, "gpstime": "2019-09-11T12:50:00+08:00"}'
r.lpush('queue.bus.gps',gps)
## 离开围栏
gps = '{ "obuid": "123456", "lat": 23.146, "lon": 113.351, "gpstime": "2019-09-11T12:50:00+08:00"}'
r.lpush('queue.bus.gps',gps)
```   

### 进围栏事件格式
- 原始围栏事件/任务ID
```
{
    "task": {
        "id": "1568048402031"
    }, 
    "command": "set", 
    "group": "5d78672f9e4ea40df84f9ee9", 
    "detect": "exit", 
    "hook": "933526:1568048402031", 
    "key": "busgps", 
    "time": "2019-09-11T11:17:03.46220216+08:00", 
    "id": "933526", 
    "object": {
        "type": "Feature", 
        "geometry": {
            "type": "Point", 
            "coordinates": [
                0, 
                0
            ]
        }, 
        "properties": {
            "fetchunix": 1568171823, 
            "gpsunix": 1568173800
        }
    }, 
    "distance": 12384563.55534404
}
```

- 围栏事件缩减
````
{
    "task": {
        "id": "87654321"
    }, 
    "detect": "enter", 
    "key": "busgps", 
    "time": "2019-09-04T15:58:27.411250342+08:00", 
    "id": "123456", 
    "object": {
        "type": "Feature", 
        "geometry": {
            "type": "Point", 
            "coordinates": [
                23.123456, 
                113.123456
            ]
        }, 
        "properties": {
            "fetchunix": 1567583907, 
            "gpsunix": 1567413000
        }
    }
}
````