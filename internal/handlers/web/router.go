package web

import "net/http"

// RegisterRoutes регистрирует статику и защищенные API эндпоинты
func (s *Server) RegisterRoutes() {
	// Serve static files
	fs := http.FileServer(http.Dir("./web/static"))
	s.mux.Handle("/", fs)

	// Wrap handlers with authMiddleware
	auth := s.authMiddleware

	// Profile
	s.mux.HandleFunc("GET /api/profile", auth(s.handleProfile))

	// Targets
	s.mux.HandleFunc("GET /api/targets", auth(s.handleGetTargets))
	s.mux.HandleFunc("POST /api/targets", auth(s.handleCreateTarget))
	s.mux.HandleFunc("GET /api/targets/{id}", auth(s.handleGetTargetByID))
	s.mux.HandleFunc("PUT /api/targets/{id}", auth(s.handleUpdateTarget))
	s.mux.HandleFunc("DELETE /api/targets/{id}", auth(s.handleDeleteTarget))

	// Keys & Monitoring
	s.mux.HandleFunc("POST /api/targets/{id}/keys", auth(s.handleAddKey))
	s.mux.HandleFunc("DELETE /api/keys/{id}", auth(s.handleDeleteKey))
	s.mux.HandleFunc("GET /api/monitoring/message", auth(s.handleCheckMessage))

	// Services
	s.mux.HandleFunc("GET /api/services", auth(s.handleGetServices))
	s.mux.HandleFunc("POST /api/services", auth(s.handleCreateService))
	s.mux.HandleFunc("GET /api/services/{id}", auth(s.handleGetServiceByID))
	s.mux.HandleFunc("PUT /api/services/{id}", auth(s.handleUpdateService))
	s.mux.HandleFunc("DELETE /api/services/{id}", auth(s.handleDeleteService))

	// Machines
	s.mux.HandleFunc("GET /api/services/{id}/machines", auth(s.handleGetMachines))
	s.mux.HandleFunc("POST /api/services/{id}/machines", auth(s.handleAddMachine))
	s.mux.HandleFunc("GET /api/services/{id}/machines/{mid}", auth(s.handleGetMachine))
	s.mux.HandleFunc("DELETE /api/services/{id}/machines/{mid}", auth(s.handleDeleteMachine))
	s.mux.HandleFunc("POST /api/services/{id}/machines/{mid}/poll", auth(s.handleStartPoll))
	s.mux.HandleFunc("DELETE /api/services/{id}/machines/{mid}/poll", auth(s.handleStopPoll))
	s.mux.HandleFunc("GET /api/services/{id}/machines/{mid}/program", auth(s.handleGetProgram))
}
