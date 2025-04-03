package application

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/dominikbraun/graph"
	incus "github.com/lxc/incus/v6/client"
	api "github.com/lxc/incus/v6/shared/api"
)

// GetIncusInstance returns the incus instance.
// If the instance is not already set, it will be set by connecting to the first service.
// If no valid remote is found, an error is returned.
//
// UseProject overwrites the instance, that means this function will return the overwritten instance.
func (app *Compose) GetIncusInstance() (incus.InstanceServer, error) {
	if app.incusInstance != nil {
		return app.incusInstance, nil
	}

	// check to see if the incus connection is valid
	// get the first service and try to connect to the incus remote
	for _, service := range app.Services {
		remote, _, err := app.conf.ParseRemote(service.Name)
		if err != nil {
			return nil, fmt.Errorf("while parsing remote: %w", err)
		}

		d, err := app.conf.GetInstanceServer(remote)
		if err != nil {
			return nil, fmt.Errorf("while getting instance server: %w", err)
		}

		app.incusInstance = d
		return d, nil
	}

	return nil, fmt.Errorf("no valid remote found")
}

// UseProject overwrites the incus instance with the given project.
// If the incus instance is not already set, it will be set by connecting to the first service.
// If no valid remote is found, an error is returned.
func (app *Compose) UseProject(name string) (incus.InstanceServer, error) {
	// get the project names while we're connected
	d, err := app.GetIncusInstance()
	if err != nil {
		return nil, fmt.Errorf("while getting instance server: %w", err)
	}

	app.incusInstance = d.UseProject(name)
	return app.incusInstance, nil
}

func (app *Compose) GetInucsProjectNames() ([]string, error) {
	if app.incusProjectNames != nil {
		return app.incusProjectNames, nil
	}

	// get the project names while we're connected
	d, err := app.GetIncusInstance()
	if err != nil {
		return nil, fmt.Errorf("while getting project names: %w", err)
	}

	projectNames, err := d.GetProjectNames()
	if err != nil {
		return nil, fmt.Errorf("while getting project names: %w", err)
	}

	app.incusProjectNames = projectNames
	return projectNames, nil
}

func (app *Compose) GetProject() string {
	switch {
	case app.ComposeProject.Name != "default":
		return app.ComposeProject.Name
	case app.Project != "":
		return app.Project
	default:
		slog.Debug("Using default project")
		return "default"
	}
}

func (app *Compose) CreateProject(name string) error {
	// get the project names while we're connected
	d, err := app.GetIncusInstance()
	if err != nil {
		return fmt.Errorf("while getting project names: %w", err)
	}

	projectNames, err := d.GetProjectNames()
	if err != nil {
		return fmt.Errorf("while getting project names: %w", err)
	}

	// check to see if the project exists
	if slices.Contains(projectNames, name) {
		return nil
	}

	// create the project
	project := api.ProjectsPost{
		Name: name,
	}
	err = d.CreateProject(project)
	if err != nil {
		return fmt.Errorf("while creating project: %w", err)
	}

	return nil
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
