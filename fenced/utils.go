package fenced

import (
	"fmt"
	"strings"

	geojson "github.com/paulmach/go.geojson"
)

//检查GPS坐标是否符合要求
func (s *Server) checkGPS(lat float64, lon float64, valid bool) bool {
	return true
}

//检查Dispatch信息是否符合要求
func (s *Server) checkMeter(enter float64, exit float64) bool {
	return true
}

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
