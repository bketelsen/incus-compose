package application

import (
	"log/slog"

	"gopkg.in/yaml.v3"
)

func (s *Service) String() string {

	bb, _ := yaml.Marshal(s)
	return string(bb)
}

func (s *Service) GetContainerName() string {
	slog.Info("GetContainerName", slog.String("container", s.ContainerName), slog.String("name", s.Name))
	if s.ContainerName != "" {
		return s.ContainerName
	}
	return s.Name
}
