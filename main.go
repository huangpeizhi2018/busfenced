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

	if b := checkFileExist(confn); !b {
		log.Info("busfenced check config file not exist", zap.String("confn", confn))
		return
	}

	var err error

	//加载分析配置文件
	cf, err := fenced.NewCf(confn)
	if err != nil {
		log.Warn("busfenced load config failure", zap.String("confn", confn), zap.Error(err))
		return
	}

	pid, err := pidfile.New(cf.PidFile)
	if err != nil {
		log.Warn("busfenced create pidfile failure", zap.String("pidfile", cf.PidFile), zap.Error(err))
		return
	}

	defer pid.Close()

	//分析服务
	server, err := fenced.NewServer(cf)
	if err != nil {
		log.Warn("busfenced server initialize failure", zap.Error(err))
		return
	}
	defer server.Close()

	if err := fenced.Run(server); err != nil {
		log.Warn("busfenced server abnormal exit", zap.Error(err))
		return
	}
}

//检查文件是否存在
func checkFileExist(filename string) bool {
	exist := true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}

	return exist
}

func init() {
	flag.Parse()

	confn = "busfenced.yaml"
}
