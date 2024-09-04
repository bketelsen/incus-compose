package application

import (
	"gopkg.in/yaml.v3"
)

func (s *Service) String() string {

	bb, _ := yaml.Marshal(s)
	return string(bb)
}
