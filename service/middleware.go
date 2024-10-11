package service

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/google/uuid"

	log "marzban-node/logger"
)

func (s *Service) checkSessionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check ip
		clientIP := s.GetIP()
		clientID := s.GetSessionID()
		if clientIP == "" || clientID == uuid.Nil {
			http.Error(w, "please connect first", http.StatusTooEarly)
			return
		}

		// check ip
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		switch {
		case err != nil:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case ip != s.GetIP():
			http.Error(w, "IP address is not valid", http.StatusForbidden)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "please connect first", http.StatusUnauthorized)
			return
		}

		tokenString := strings.Split(authHeader, " ")[1]
		// check session id
		sessionID, err := uuid.Parse(tokenString)
		switch {
		case err != nil:
			http.Error(w, "please send valid uuid", http.StatusUnprocessableEntity)
			return
		case sessionID != clientID:
			http.Error(w, "session id mismatch.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		logMessage := fmt.Sprintf("%s, %s, %s, %d", r.RemoteAddr, r.Method, r.URL.Path, ww.Status())
		log.Api(logMessage)
	})
}
