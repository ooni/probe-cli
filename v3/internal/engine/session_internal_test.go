package engine

import (
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func (s *Session) SetAssetsDir(assetsDir string) {
	s.assetsDir = assetsDir
}

func (s *Session) GetAvailableProbeServices() []model.Service {
	return s.getAvailableProbeServices()
}

func (s *Session) AppendAvailableProbeService(svc model.Service) {
	s.availableProbeServices = append(s.availableProbeServices, svc)
}

func (s *Session) QueryProbeServicesCount() int64 {
	return s.queryProbeServicesCount.Load()
}
