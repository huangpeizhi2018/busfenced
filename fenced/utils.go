package fenced

import (
	"fmt"
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
