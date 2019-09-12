# 车辆GPS数据分析围栏的进出事件配置文件说明

## 服务名
···
service: busfenced-cmder
···

## 全局配置
gpstimeoffset: -1
chanlen: 10
pidfile: /opt/busfenced/pid/busfenced.pid

## 数据源，含全量出租车GPS数据源与调度应用事件
source:
  addr: 127.0.0.1
  port: 6390
  passwd:
  maxidle: 5
  gpspoint: queue.bus.gps
  gpstouch: touch.bus.gps
  dispatchpoint: queue.bus.dispatch
  dispatchtouch: touch.bus.dispatch

# 输出事件REDIS
target:
  addr: 127.0.0.1
  port: 6390
  passwd:
  maxidle: 5
  enterpoint: event.busfence.enter
  entertouch: touch.busfence.enter
  exitpoint: event.busfence.exit
  exittouch: touch.busfence.exit

# SETHOOK HOOK名称 redis://10.0.20.78:6379/pub-point NEARBY GPS集合 FENCE DETECT enter,exit 中心点经度 中心点纬度 米
# 其中，GPS集合、中心点经度、中心点纬度、米，从数据库或静态文件中获取。
enterfenced:
  cmd: /opt/busfenced/bin/tile38-server
  homedir: /opt/busfenced/tile38/enterfence
  clean: true
  addr: 127.0.0.1
  port: 7875
  collection: busgps
  pubpoint: redis://127.0.0.1:6390/pub-enterfenced

exitfenced:
  cmd: /opt/busfenced/bin/tile38-server
  homedir: /opt/busfenced/tile38/exitfence
  clean: true
  addr: 127.0.0.1
  port: 7876
  collection: busgps
  pubpoint: redis://127.0.0.1:6390/pub-exitfenced

# 自动内存清理
aofshrink:
  seconds: 3600
  valid: false

# 日志配置
zlog:
  level: debug
  development: true
  encoding: console
  outputPaths: ['/opt/busfenced/log/busfenced.log', stdout]
  errorOutputPaths: [stderr]

stats:
  addr: 0.0.0.0
  port: 9875
  valid: true