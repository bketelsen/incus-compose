package application

import (
	"gopkg.in/yaml.v3"
)

func (s *Service) String() string {

	bb, _ := yaml.Marshal(s)
	return string(bb)
}

func (s *Service) GetContainerName() string {
	if s.ContainerName != "" {
		return s.ContainerName
	}
	return s.Name
}
