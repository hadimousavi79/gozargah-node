package rest

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/m03ed/gozargah-node/backend"
	"github.com/m03ed/gozargah-node/backend/xray"
	"github.com/m03ed/gozargah-node/common"
)

func (s *Service) Base(w http.ResponseWriter, _ *http.Request) {
	common.SendProtoResponse(w, s.controller.BaseInfoResponse(false, ""))
}

func (s *Service) Start(w http.ResponseWriter, r *http.Request) {
	ctx, backendType, err := s.detectBackend(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "unknown ip", http.StatusServiceUnavailable)
		return
	}

	if s.controller.GetBackend() != nil {
		log.Println("New connection from ", ip, " core control access was taken away from previous client.")
		s.disconnect()
	}

	s.connect(ip)

	log.Println(ip, " connected, Session ID = ", s.controller.GetSessionID())

	if err = s.controller.StartBackend(ctx, backendType); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	common.SendProtoResponse(w, s.controller.BaseInfoResponse(true, ""))
}

func (s *Service) Stop(w http.ResponseWriter, _ *http.Request) {
	log.Println(s.GetIP(), " disconnected, Session ID = ", s.controller.GetSessionID())
	s.disconnect()

	common.SendProtoResponse(w, &common.Empty{})
}

func (s *Service) detectBackend(r *http.Request) (context.Context, common.BackendType, error) {
	var data common.Backend
	var ctx context.Context

	if err := common.ReadProtoBody(r.Body, &data); err != nil {
		return nil, 0, err
	}

	if data.Type == common.BackendType_XRAY {
		config, err := xray.NewXRayConfig(data.Config)
		if err != nil {
			return nil, 0, err
		}
		ctx = context.WithValue(r.Context(), backend.ConfigKey{}, config)
	} else {
		return ctx, data.Type, errors.New("invalid backend type")
	}

	ctx = context.WithValue(ctx, backend.UsersKey{}, data.GetUsers())

	return ctx, data.Type, nil
}
