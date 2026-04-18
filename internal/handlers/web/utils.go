package web

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func parseID(r *http.Request, key string) (uint, error) {
	val, err := strconv.ParseUint(r.PathValue(key), 10, 32)
	return uint(val), err
}

func getUserID(r *http.Request) int64 {
	return r.Context().Value("userID").(int64)
}
