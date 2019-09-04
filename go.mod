module github.com/huangpeizhi2018/busfenced

go 1.12

replace github.com/huangpeizhi2018/busfenced/fenced => ./fenced

replace github.com/huangpeizhi2018/busfenced/fenced/version => ./fenced/version

require (
	github.com/ReneKroon/ttlcache v1.5.0
	github.com/gomodule/redigo v1.7.0
	github.com/influxdata/pidfile v0.0.0-20171020183418-16df69ba8e75
	github.com/paulmach/go.geojson v1.4.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/stretchr/testify v1.4.0 // indirect
	github.com/tidwall/gjson v1.3.2
	github.com/tidwall/sjson v1.0.4
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0
	gopkg.in/yaml.v2 v2.2.2
)
