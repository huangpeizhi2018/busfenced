package fenced

//检查GPS坐标是否符合要求
func (s *Server) checkGPS(lat float64, lon float64, valid bool) bool {
	return true
}

//检查Dispatch信息是否符合要求
func (s *Server) checkMeter(enter float64, exit float64) bool {
	return true
}

