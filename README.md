# busfenced/公交总站围栏处理

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
    "enterMeter": 100, 
    "exitMeter": 200, 
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
dispatch = '{"obuId": "933526","lat": 23.145199,"lon": 113.35396,"enterMeter": 200,"exitMeter": 200,"taskId": "1568048402031","invalidTime": "2019-08-29T17:30:00+08:00"}'
r.lpush('queue.bus.dispatch',dispatch)
# GPS信息  
gps = '{ "obuid": "933526", "lat": 23.1463889, "lon": 113.3525000, "gpstime": "2019-09-10T11:50:00+08:00"}'
r.lpush('queue.bus.gps',gps)
```   

### 进围栏事件
- 原始围栏事件/任务ID
````
{
    "task": {
        "id": "87654321"
    }, 
    "command": "set", 
    "group": "5d6f6ea39e4ea469b438e7c8", 
    "detect": "enter", 
    "hook": "123456:87654321", 
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