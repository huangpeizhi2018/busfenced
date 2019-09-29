package fenced

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

//配置结构
type Conf struct {
	ConfigFile        string    `yaml:"-"`
	Service           string    `yaml:"service"`           //服务名称
	GpsTimeOffset     float64   `yaml:"gpstimeoffset"`     //GPSTIME偏移秒数，不管偏移值为-1
	InvalidTimeOffset int64     `yaml:"invalidtimeoffset"` //为防止“业务系统的调度过期时间”过快或过慢，添加适当的偏移秒数。
	ChanLen           int       `yaml:"chanlen"`
	PidFile           string    `yaml:"pidfile"`
	Source            Source    `yaml:"source"`
	Target            Target    `yaml:"target"`
	EnterFenced       Fenced    `yaml:"enterfenced"`
	ExitFenced        Fenced    `yaml:"exitfenced"`
	AOFShrink         AOFShrink `yaml:"aofshrink"`
	Stats             *Stats    `yaml:"stats"`
	ZLog              ZLog      `yaml:"zlog"`
}

//来源数据
type Source struct {
	Addr          string `yaml:"addr"`
	Port          string `yaml:"port"`
	Passwd        string `yaml:"passwd"`
	MaxIdel       int    `yaml:"maxidle"`
	GPSPoint      string `yaml:"gpspoint"`
	GPSTouch      string `yaml:"gpstouch"`
	DispatchPoint string `yaml:"dispatchpoint"`
	DispatchTouch string `yaml:"dispatchtouch"`
}

//输出事件
type Target struct {
	Addr       string `yaml:"addr"`
	Port       string `yaml:"port"`
	Passwd     string `yaml:"passwd"`
	MaxIdel    int    `yaml:"maxidle"`
	EnterPoint string `yaml:"enterpoint"`
	EnterTouch string `yaml:"entertouch"`
	ExitPoint  string `yaml:"exitpoint"`
	ExitTouch  string `yaml:"exittouch"`
}

//进入围栏TILE38服务
type Fenced struct {
	Cmd        string `yaml:"cmd"`
	HomeDir    string `yaml:"homedir"`
	Clean      bool   `yaml:"clean"`
	Addr       string `yaml:"addr"`
	Port       string `yaml:"port"`
	Collection string `yaml:"collection"`
	PubPoint   string `yaml:"pubpoint"`
	DeleteNow  bool   `yaml:"deletenow"` //是否触发事件后，立即清除HOOK。
	Distance   int64  `yaml:"distance"`  //围栏事件触发有效的距离定义。
}

//AOFSHRINK服务
type AOFShrink struct {
	Seconds int64 `yaml:"seconds"`
	Valid   bool  `yaml:"valid"`
}

//性能指标
type Stats struct {
	Addr  string `yaml:"addr"`
	Port  string `yaml:"port"`
	Valid bool   `yaml:"valid"`
}

//日志配置
type ZLog struct {
	Level            string   `yaml:"level"`
	Development      bool     `yaml:"development"`
	Encoding         string   `yaml:"encoding"`
	OutputPaths      []string `yaml:"outputPaths"`
	ErrorOutputPaths []string `yaml:"errorOutputPaths"`
}

//加载配置
func NewCf(filename string) (*Conf, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer func() {
		if f != nil {
			_ = f.Close()

		}
	}()

	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var c = &Conf{}
	err = c.load(bs)
	if err != nil {
		return nil, err
	}

	dir, err := filepath.Abs(f.Name())
	if err != nil {
		return nil, err
	}
	c.ConfigFile = filepath.Join(dir, filepath.FromSlash(path.Clean("/"+f.Name())))

	return c, nil
}

func (c *Conf) load(bs []byte) error {
	err := yaml.Unmarshal(bs, c)
	if err != nil {
		return err
	}

	err = c.verify()
	if err != nil {
		return err
	}

	return nil
}

//校验配置文件的正确性
func (c *Conf) verify() error {
	if c.Service == "" {
		return fmt.Errorf("service no value")
	}

	pf := strings.TrimSpace(c.PidFile)
	if pf == "" || pf == "/" {
		return fmt.Errorf(fmt.Sprintf("pidfile %s configure invalid", pf))
	}

	if len(c.Stats.Addr) == 0 || len(c.Stats.Port) == 0 {
		return fmt.Errorf("stats config invalid")
	}

	if c.ChanLen == 0 {
		c.ChanLen = 100
	}

	return nil
}

//生成配置文件
func (c *Conf) Save(out io.Writer) error {
	bs, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	_, err = out.Write(bs)
	if err != nil {
		return err
	}
	return nil
}
