package application

import (
	"log/slog"
	"slices"

	"github.com/dominikbraun/graph"
)

func Generate(path string) error {

	return nil
}

func (app *Compose) GetProject() string {
	if app.Project == "" {
		slog.Debug("Using default project")
		return "default"
	}
	return app.Project
}

func (app *Compose) GetProfiles() []string {
	if len(app.Profiles) == 0 {
		slog.Debug("Using default profiles")
		return []string{"default"}
	}
	return app.Profiles
}

// Order returns the order in which services should be started or stopped.
// If reverse is true, the order is reversed.
// Use reverse=true for starting services.
// Use reverse=false for stopping services.
func (app *Compose) Order(reverse bool) []string {
	if app.Dag != nil {
		order, _ := graph.TopologicalSort(app.Dag)
		if reverse {
			slices.Reverse(order)
		}
		return order
	}
	return []string{}
}
