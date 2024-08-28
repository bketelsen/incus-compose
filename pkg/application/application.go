package application

import (
	"log/slog"
	"os"
	"slices"

	"github.com/dominikbraun/graph"
	"gopkg.in/yaml.v3"
)

var defaultNames = []string{"incus-compose.yaml", "incus-compose.yml"}

type Compose struct {
	Name       string                      `yaml:"name" validate:"required"`
	Project    string                      `yaml:"project,omitempty" validate:"project-exists"`
	Services   map[string]Service          `yaml:"services" validate:"required,dive,required"`
	Profiles   []string                    `yaml:"profiles" validate:"dive,profile-exists"`
	ExportPath string                      `yaml:"export_path,omitempty"`
	dag        graph.Graph[string, string] `yaml:"-"`
}

type Service struct {
	Image                 string            `yaml:"image" validate:"required"`
	GPU                   bool              `yaml:"gpu,omitempty"`
	Volumes               map[string]Volume `yaml:"volumes,omitempty" validate:"dive,required"`
	BindMounts            map[string]Bind   `yaml:"binds,omitempty"`
	AdditionalProfiles    []string          `yaml:"additional_profiles,omitempty" validate:"dive,profile-exists"`
	EnvironmentFile       string            `yaml:"environment_file,omitempty"`
	CloudInitUserData     string            `yaml:"cloud_init_user_data,omitempty"`
	CloudInitUserDataFile string            `yaml:"cloud_init_user_data_file,omitempty"`
	Snapshot              Snapshot          `yaml:"snapshot,omitempty"`
	DependsOn             []string          `yaml:"depends_on,omitempty"`
	InventoryGroups       []string          `yaml:"inventory_groups,omitempty"`
}

type Snapshot struct {
	Schedule string `yaml:"schedule,omitempty"`
	Expiry   string `yaml:"expiry,omitempty"`
	Pattern  string `yaml:"pattern,omitempty"`
}

type Volume struct {
	Mountpoint string   `yaml:"mountpoint"`
	Pool       string   `yaml:"pool" validate:"pool-exists"`
	Snapshot   Snapshot `yaml:"snapshot,omitempty"`
}

type Bind struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Shift  bool   `yaml:"shift,omitempty"`
}

func Load(workdir, path string) (Compose, error) {

	app := Compose{}
	if path == "" {
		slog.Debug("Searching for compose file")
		path = firstExistingFile(defaultNames)
	}

	slog.Debug("Using compose file", slog.String("path", path))
	data, err := os.ReadFile(path)
	if err != nil {
		return app, err
	}
	err = yaml.Unmarshal(data, &app)
	if err != nil {
		return app, err
	}
	g := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic())
	for name := range app.Services {
		_ = g.AddVertex(name)
	}
	for name := range app.Services {
		for _, dep := range app.Services[name].DependsOn {
			_ = g.AddEdge(name, dep)
		}
	}
	app.dag = g

	err = app.Validate()
	if err != nil {
		return app, err
	}
	return app, nil
}

func firstExistingFile(files []string) string {
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}
	return ""
}

func Generate(path string) error {
	app := Compose{}
	app.Name = "testapp"
	app.Project = "testproject"
	app.Profiles = []string{"default", "profile1"}

	service := Service{}
	service.Image = "images:ubuntu/noble/cloud"
	service.GPU = false
	service.AdditionalProfiles = []string{"profile2"}
	service.CloudInitUserDataFile = "testapp.yaml"
	service.DependsOn = []string{"dbservice"}

	volume := Volume{}
	volume.Mountpoint = "/metadata"
	volume.Pool = "slowpool"

	bind := Bind{}
	bind.Type = "disk"
	bind.Source = "/mnt/media"
	bind.Target = "/media"
	bind.Shift = true

	service.Volumes = map[string]Volume{
		"metadatavolume": volume,
	}
	service.BindMounts = map[string]Bind{
		"mediabind": bind,
	}
	service.Snapshot = Snapshot{
		Schedule: "@daily",
		Expiry:   "14d",
	}

	service2 := Service{}
	service2.Image = "docker:postgres"

	volume2 := Volume{}
	volume2.Mountpoint = "/data"
	volume2.Pool = "fast"
	volume2.Snapshot = Snapshot{
		Schedule: "@hourly",
		Expiry:   "7d",
	}
	service2.Volumes = map[string]Volume{
		"data": volume2,
	}
	app.Services = map[string]Service{
		"testservice": service,
		"dbservice":   service2,
	}
	data, err := yaml.Marshal(app)
	if err != nil {
		return err

	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
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
	order, _ := graph.TopologicalSort(app.dag)
	if reverse {
		slices.Reverse(order)
	}
	return order
}
