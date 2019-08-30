#!/usr/bin/env bash

# 启动围栏处理服务
nohup /opt/busfenced/bin/tile38-server -p 7875 -d /opt/busfenced/tile38/enterfence/ -vv --appendonly no --dev --pidfile /opt/busfenced/pid/tile38_7875.pid >> /opt/busfenced/log/tile38_7875.log &
nohup /opt/busfenced/bin/tile38-server -p 7876 -d /opt/busfenced/tile38/exitfence/ -vv --appendonly no --dev --pidfile /opt/busfenced/pid/tile38_7876.pid >> /opt/busfenced/log/tile38_7876.log &
# 启动Redis
/usr/local/bin/redis-server /opt/redis/conf/redis_6390.conf
# 启动公交围栏事件服务
nohup /opt/busfenced/bin/busfenced /opt/busfenced/conf/busfenced.yaml &

