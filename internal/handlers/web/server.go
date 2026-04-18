package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/iwtcode/fanucClient/internal/interfaces"
	"github.com/iwtcode/fanucService"
)

type Server struct {
	mux        *http.ServeMux
	server     *http.Server
	settingsUC interfaces.SettingsUsecase
	monitorUC  interfaces.MonitoringUsecase
	controlUC  interfaces.ControlUsecase
	sseBroker  *SSEBroker
}

func NewServer(sUC interfaces.SettingsUsecase, mUC interfaces.MonitoringUsecase, cUC interfaces.ControlUsecase) *Server {
	s := &Server{
		mux:        http.NewServeMux(),
		settingsUC: sUC,
		monitorUC:  mUC,
		controlUC:  cUC,
		sseBroker:  NewSSEBroker(),
	}
	s.RegisterRoutes()
	return s
}

// BroadcastToUser реализация интерфейса WebSender для Notifier Service
func (s *Server) BroadcastToUser(userID int64, eventType string, data []byte) {
	s.sseBroker.BroadcastToUser(userID, eventType, data)
}

func (s *Server) Start(port string) {
	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: s.mux,
	}
	log.Printf("🌐 Web-сервер запущен на http://localhost:%s", port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Web server error: %v", err)
	}
}

func (s *Server) Stop(ctx context.Context) {
	if s.server != nil {
		s.server.Shutdown(ctx)
	}
}

// ... остальной код серверных обработчиков (handleProfile, handleGetTargets и т.д.) остается БЕЗ ИЗМЕНЕНИЙ ...
func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	targets, _ := s.settingsUC.GetTargets(userID)
	services, _ := s.settingsUC.GetServices(userID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"targets_count":  len(targets),
		"services_count": len(services),
	})
}

func (s *Server) handleGetTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := s.settingsUC.GetTargets(getUserID(r))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, targets)
}

func (s *Server) handleGetTargetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	target, err := s.settingsUC.GetTargetByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "target not found")
		return
	}
	respondJSON(w, http.StatusOK, target)
}

func (s *Server) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Broker string `json:"broker"`
		Topic  string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.settingsUC.CreateTargetDirect(getUserID(r), req.Name, req.Broker, req.Topic); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleUpdateTarget(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Name   string `json:"name"`
		Broker string `json:"broker"`
		Topic  string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.settingsUC.UpdateTargetDirect(getUserID(r), id, req.Name, req.Broker, req.Topic); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err == nil {
		s.settingsUC.DeleteTarget(getUserID(r), id)
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAddKey(w http.ResponseWriter, r *http.Request) {
	tid, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.settingsUC.AddKeyToTargetDirect(tid, req.Key); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err == nil {
		s.settingsUC.DeleteKey(id)
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCheckMessage(w http.ResponseWriter, r *http.Request) {
	tidStr := r.URL.Query().Get("targetId")
	kidStr := r.URL.Query().Get("keyId")
	tid, _ := strconv.ParseUint(tidStr, 10, 32)
	kid, _ := strconv.ParseUint(kidStr, 10, 32)

	keyName, val, err := s.monitorUC.FetchLastKafkaMessage(r.Context(), uint(tid), uint(kid))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(val), &parsed); err == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"key": keyName, "data": parsed})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"key": keyName, "data": val})
}

func (s *Server) handleGetServices(w http.ResponseWriter, r *http.Request) {
	services, err := s.settingsUC.GetServices(getUserID(r))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, services)
}

func (s *Server) handleGetServiceByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	svc, err := s.settingsUC.GetServiceByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "not found")
		return
	}
	respondJSON(w, http.StatusOK, svc)
}

func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.settingsUC.CreateServiceDirect(getUserID(r), req.Name, req.BaseURL, req.APIKey); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Name    string `json:"name"`
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.settingsUC.UpdateServiceDirect(getUserID(r), id, req.Name, req.BaseURL, req.APIKey); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err == nil {
		s.settingsUC.DeleteService(getUserID(r), id)
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetMachines(w http.ResponseWriter, r *http.Request) {
	sid, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	machines, err := s.controlUC.ListMachines(r.Context(), sid)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, machines)
}

func (s *Server) handleGetMachine(w http.ResponseWriter, r *http.Request) {
	sid, _ := parseID(r, "id")
	mid := r.PathValue("mid")
	m, err := s.controlUC.GetMachine(r.Context(), sid, mid)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, m)
}

func (s *Server) handleAddMachine(w http.ResponseWriter, r *http.Request) {
	sid, _ := parseID(r, "id")
	var req fanucService.ConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}
	m, err := s.controlUC.CreateMachine(r.Context(), sid, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, m)
}

func (s *Server) handleDeleteMachine(w http.ResponseWriter, r *http.Request) {
	sid, _ := parseID(r, "id")
	mid := r.PathValue("mid")
	if err := s.controlUC.DeleteMachine(r.Context(), sid, mid); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStartPoll(w http.ResponseWriter, r *http.Request) {
	sid, _ := parseID(r, "id")
	mid := r.PathValue("mid")
	var req struct {
		Interval int `json:"interval"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Interval < 100 {
		req.Interval = 5000
	}

	if err := s.controlUC.StartPolling(r.Context(), sid, mid, req.Interval); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStopPoll(w http.ResponseWriter, r *http.Request) {
	sid, _ := parseID(r, "id")
	mid := r.PathValue("mid")
	if err := s.controlUC.StopPolling(r.Context(), sid, mid); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetProgram(w http.ResponseWriter, r *http.Request) {
	sid, _ := parseID(r, "id")
	mid := r.PathValue("mid")
	prog, err := s.controlUC.GetProgram(r.Context(), sid, mid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(prog))
}
