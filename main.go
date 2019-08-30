package main

import (
	"flag"
	"os"

	"github.com/huangpeizhi2018/busfenced/fenced"
	"github.com/influxdata/pidfile"
	"go.uber.org/zap"
)

var confn string    //配置文件名
var log *zap.Logger //日志

//主程序
func main() {
	log, _ = zap.NewDevelopment()

	args := flag.Args()
	if le := len(args); le >= 1 {
		confn = args[0]
	}
	log.Info("runcmd", zap.String("prog", os.Args[0]), zap.String("confn", confn))

	var err error

	//加载分析配置文件
	cf, err := fenced.NewCf(confn)
	if err != nil {
		log.Warn("busfenced load config", zap.String("confn", confn), zap.Error(err))
		return
	}

	pid, err := pidfile.New(cf.PidFile)
	if err != nil {
		log.Warn("busfenced create pid file", zap.String("pidfile", cf.PidFile), zap.Error(err))
		return
	}

	defer pid.Close()

	//分析服务
	server, err := fenced.NewServer(cf)
	if err != nil {
		log.Warn("busfenced service initialize", zap.Error(err))
		return
	}
	defer server.Close()

	if err := server.Run(); err != nil {
		log.Warn("busfenced service run", zap.Error(err))
		return
	}
}

func init() {
	flag.Parse()

	confn = "cmder.yaml"
}
