package application

import (
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/dominikbraun/graph"
	config "github.com/lxc/incus/v6/shared/cliconfig"
)

type Compose struct {
	Name           string                      `yaml:"name" validate:"required"`
	Project        string                      `yaml:"project,omitempty" validate:"project-exists"`
	Services       map[string]Service          `yaml:"services" validate:"required,dive,required"`
	Profiles       []string                    `yaml:"profiles" validate:"dive,profile-exists"`
	ExportPath     string                      `yaml:"export_path,omitempty"`
	Dag            graph.Graph[string, string] `yaml:"-"`
	ComposeProject *types.Project              `yaml:"-"`
	SecretsFiles   map[string]SecretsFile      `yaml:"secretsfiles,omitempty"`
	conf           *config.Config
}

type Service struct {
	Name                  string             `yaml:"name" validate:"required"`
	ContainerName         string             `yaml:"containername,omitempty"`
	Image                 string             `yaml:"image" validate:"required"`
	GPU                   bool               `yaml:"gpu,omitempty"`
	Volumes               map[string]*Volume `yaml:"volumes,omitempty" validate:"dive,required"`
	BindMounts            map[string]Bind    `yaml:"binds,omitempty"`
	AdditionalProfiles    []string           `yaml:"additional_profiles,omitempty" validate:"dive,profile-exists"`
	EnvironmentFile       string             `yaml:"environment_file,omitempty"`
	Environment           map[string]*string `yaml:"environment,omitempty"`
	CloudInitUserData     string             `yaml:"cloud_init_user_data,omitempty"`
	CloudInitUserDataFile string             `yaml:"cloud_init_user_data_file,omitempty"`
	Snapshot              *Snapshot          `yaml:"snapshot,omitempty"`
	DependsOn             []string           `yaml:"depends_on,omitempty"`
	InventoryGroups       []string           `yaml:"inventory_groups,omitempty"`
	Storage               string             `yaml:"storage,omitempty"`
	Secrets               map[string]Secret  `yaml:"secrets,omitempty"`
}

type Snapshot struct {
	Schedule string `yaml:"schedule,omitempty"`
	Expiry   string `yaml:"expiry,omitempty"`
	Pattern  string `yaml:"pattern,omitempty"`
}

type Volume struct {
	Name       string    `yaml:"name,omitempty"`
	Mountpoint string    `yaml:"mountpoint"`
	Pool       string    `yaml:"pool" validate:"pool-exists"`
	Snapshot   *Snapshot `yaml:"snapshot,omitempty"`
}

type Bind struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Shift  bool   `yaml:"shift,omitempty"`
}

type Secret struct {
	MountPoint string `yaml:"filepath,omitempty"`
}

type SecretsFile struct {
	FilePath string `yaml:"filepath,omitempty"`
}
