#!/usr/bin/env bash

kill -9 `cat /opt/busfenced/pid/busfenced.pid` || rm -f /opt/busfenced/pid/busfenced.pid
kill -9 `cat /opt/busfenced/pid/tile38_7875.pid` || rm -f /opt/busfenced/pid/tile38_7875.pid
kill -9 `cat /opt/busfenced/pid/tile38_7875.pid` || rm -f /opt/busfenced/pid/tile38_7876.pid