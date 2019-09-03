package fenced

func Run(server *Server) error {
	err := server.Run()
	return err
}
