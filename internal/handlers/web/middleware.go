package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
)

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-User-Id")
		if userIDStr == "" {
			// Читаем из query-параметра (для поддержки EventSource/SSE)
			userIDStr = r.URL.Query().Get("uid")
		}

		if userIDStr == "" {
			respondError(w, http.StatusUnauthorized, "Missing X-User-Id header or uid query param")
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil || userID >= 0 {
			respondError(w, http.StatusBadRequest, "Invalid Web User ID (must be negative)")
			return
		}

		// Проверяем, существует ли пользователь, если нет - создаем
		user, _ := s.settingsUC.GetUser(userID)
		if user == nil {
			err := s.settingsUC.RegisterUser(&entities.User{
				ID:        userID,
				FirstName: "WebUser",
				UserName:  fmt.Sprintf("web_%d", userID),
				State:     entities.StateIdle,
			})
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to register web user")
				return
			}
		}

		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
