package fenced

import (
	"fmt"
	"go.uber.org/zap"
	"strings"

	geojson "github.com/paulmach/go.geojson"
)

//检查GPS坐标是否符合要求
func (s *Server) checkGPS(lat float64, lon float64) bool {
	var ret = true

	//lat-纬度
	//lon-经度
	if lat < 20 || lon < 110 || lat > 30 || lon > 120 {
		ret = false
		s.log.Debug("lat/lon invalid", zap.Float64("lat", lat), zap.Float64("lon", lon))
	}

	return ret
}

//检查Dispatch信息是否符合要求
func (s *Server) checkMeter(meter float64) bool {
	return true
}

//gps点转换为geojson对象格式，并添加“附加属性”
func (s *Server) mkGeojson(ty string, i GPS) ([]byte, error) {
	var g *geojson.Geometry

	switch ty {
	case "POINT":
		g = geojson.NewPointGeometry([]float64{i.Lon, i.Lat})
	case "BOUNDS":
		g = geojson.NewPolygonGeometry([][][]float64{{{i.Lon, i.Lat}, {i.Lon, i.Lat}, {i.Lon, i.Lat}, {i.Lon, i.Lat}}})
	default:
		return nil, fmt.Errorf("mkGeojson fence type %s error", ty)
	}

	f := geojson.NewFeature(g)
	f.SetProperty("gpsunix", i.GPSUnix)
	f.SetProperty("fetchunix", i.FetchUnix)

	return f.MarshalJSON()
}

//HOOK格式， id:taskid
func parseHook(h string) (id string, taskid string, valid bool) {
	ph := strings.Split(h, ":")
	if len(ph) != 2 {
		return "", "", false
	}

	return ph[0], ph[1], true
}
