package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

type IncusClient struct {
	client incus.InstanceServer
}

func NewIncusClient() (*IncusClient, error) {
	c, err := incus.ConnectIncusUnix("/var/lib/incus/unix.socket", nil)
	if err != nil {
		c, err = incus.ConnectIncusUnix("/run/incus/unix.socket", nil)
		if err != nil {
			return nil, err
		}
	}

	return &IncusClient{client: c}, nil
}

func (i *IncusClient) WithProject(project string) {
	i.client = i.client.UseProject(project)
}

func (i *IncusClient) GetProjectNames() ([]string, error) {
	return i.client.GetProjectNames()
}

func (i *IncusClient) GetProfileNames() ([]string, error) {
	return i.client.GetProfileNames()
}

func (i *IncusClient) GetStoragePoolNames() ([]string, error) {
	return i.client.GetStoragePoolNames()
}

func (i *IncusClient) GetInstance(name string) (*api.Instance, string, error) {
	return i.client.GetInstance(name)
}
func (i *IncusClient) GetInstanceState(name string) (*api.InstanceState, string, error) {
	return i.client.GetInstanceState(name)
}

func (i *IncusClient) CreateProfile(name string, data api.ProfilePut) error {
	// Create the profile
	profile := api.ProfilesPost{}
	profile.Name = name
	profile.ProfilePut = data

	err := i.client.CreateProfile(profile)
	if err != nil {
		return err
	}

	return nil

}
func (i *IncusClient) DeleteProfile(name string) error {
	return i.client.DeleteProfile(name)
}

// 		args := []string{"config", "device", "add", service, bindName, bind.Type, "source=" + bind.Source, "path=" + bind.Target}

func (i *IncusClient) AddDevice(instance, name string, device map[string]string) error {

	inst, etag, err := i.client.GetInstance(instance)
	if err != nil {
		return err
	}

	_, ok := inst.Devices[name]
	if ok {
		return errors.New("device already exists")
	}

	inst.Devices[name] = device

	op, err := i.client.UpdateInstance(instance, inst.Writable(), etag)
	if err != nil {
		return err
	}

	err = op.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (i *IncusClient) SnapshotInstance(instanceName, snapshotName string, stateful bool, noexpiry bool, expiration time.Time) error {

	req := api.InstanceSnapshotsPost{
		Name:     snapshotName,
		Stateful: stateful,
	}

	if noexpiry {
		req.ExpiresAt = &time.Time{}
	} else if !expiration.IsZero() {
		req.ExpiresAt = &expiration
	}

	op, err := i.client.CreateInstanceSnapshot(instanceName, req)
	if err != nil {
		return err
	}

	return op.Wait()

}

func (i *IncusClient) SnapshotVolume(pool, volume, snapshotName string, stateful bool, noexpiry bool, expiration time.Time) error {

	req := api.StorageVolumeSnapshotsPost{
		Name: snapshotName,
	}

	if noexpiry {
		req.ExpiresAt = &time.Time{}
	} else if !expiration.IsZero() {
		req.ExpiresAt = &expiration
	}

	op, err := i.client.CreateStoragePoolVolumeSnapshot(pool, "custom", volume, req)
	if err != nil {
		return err
	}

	return op.Wait()

}

func (i *IncusClient) ExportInstance(instanceName, targetName string, instanceOnly bool) error {

	req := api.InstanceBackupsPost{
		Name:             "",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		InstanceOnly:     instanceOnly,
		OptimizedStorage: false,
	}

	op, err := i.client.CreateInstanceBackup(instanceName, req)
	if err != nil {
		return fmt.Errorf("create instance backup: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return err
	}
	// Get name of backup
	uStr := op.Get().Resources["backups"][0]
	u, err := url.Parse(uStr)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", uStr, err)
	}

	backupName, err := url.PathUnescape(path.Base(u.EscapedPath()))
	if err != nil {
		return fmt.Errorf("invalid backup name segment in path %q: %w", u.EscapedPath(), err)
	}

	defer func() {
		// Delete backup after we're done
		op, err = i.client.DeleteInstanceBackup(instanceName, backupName)
		if err == nil {
			_ = op.Wait()
		}
	}()

	var target *os.File

	target, err = os.Create(targetName)
	if err != nil {
		return err
	}

	defer func() { _ = target.Close() }()

	backupFileRequest := incus.BackupFileRequest{
		BackupFile: io.WriteSeeker(target),
	}
	_, err = i.client.GetInstanceBackupFile(instanceName, backupName, &backupFileRequest)
	if err != nil {
		_ = os.Remove(targetName)
		return fmt.Errorf("fetch instance backup file: %w", err)
	}
	err = target.Close()
	if err != nil {
		return fmt.Errorf("failed to close export file: %w", err)
	}

	return nil
}
func (i *IncusClient) ExportVolume(pool, volume, targetName string) error {

	req := api.StoragePoolVolumeBackupsPost{
		Name:             "",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		VolumeOnly:       true,
		OptimizedStorage: false,
	}

	op, err := i.client.CreateStoragePoolVolumeBackup(pool, volume, req)
	if err != nil {
		return fmt.Errorf("failed to create storage volume backup: %w", err)
	}

	err = op.Wait()
	if err != nil {
		return err
	}

	// Get name of backup
	uStr := op.Get().Resources["backups"][0]
	u, err := url.Parse(uStr)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", uStr, err)
	}

	backupName, err := url.PathUnescape(path.Base(u.EscapedPath()))
	if err != nil {
		return fmt.Errorf("invalid backup name segment in path %q: %w", u.EscapedPath(), err)
	}

	defer func() {
		// Delete backup after we're done
		op, err = i.client.DeleteStoragePoolVolumeBackup(pool, volume, backupName)
		if err == nil {
			_ = op.Wait()
		}
	}()

	target, err := os.Create(targetName)
	if err != nil {
		return err
	}

	defer func() { _ = target.Close() }()

	backupFileRequest := incus.BackupFileRequest{
		BackupFile: io.WriteSeeker(target),
	}

	// Export tarball
	_, err = i.client.GetStoragePoolVolumeBackupFile(pool, volume, backupName, &backupFileRequest)
	if err != nil {
		_ = os.Remove(targetName)
		return fmt.Errorf("failed to fetch storage volume backup file: %w", err)
	}

	return nil
}

// InstanceAction performs an action on an instance
// valid actions are: start, stop, pause, resume
// stateful is used to indicate that the instance should be stopped in a stateful way
// force is used to indicate that the instance should be stopped forcefully
func (i *IncusClient) InstanceAction(action, instance string, stateful, force bool, timeout int) error {
	state := false

	// Pause is called freeze
	if action == "pause" {
		action = "freeze"
	}

	// Resume is called unfreeze
	if action == "resume" {
		action = "unfreeze"
	}

	// Only store state if asked to
	if action == "stop" && stateful {
		state = true
	}

	if action == "start" {
		current, _, err := i.client.GetInstance(instance)
		if err != nil {
			return err
		}

		// "start" for a frozen instance means "unfreeze"
		if current.StatusCode == api.Frozen {
			action = "unfreeze"
		}

		// Always restore state (if present) unless asked not to
		if action == "start" && current.Stateful && stateful {
			state = true
		}
	}

	req := api.InstanceStatePut{
		Action:   action,
		Timeout:  20, // TODO: make this configurable
		Force:    force,
		Stateful: state,
	}

	op, err := i.client.UpdateInstanceState(instance, req, "")
	if err != nil {
		return err
	}

	return op.Wait()

}

func (i *IncusClient) DeleteInstance(instance string) error {
	op, err := i.client.DeleteInstance(instance)
	if err != nil {
		return err
	}

	return op.Wait()
}

func (i *IncusClient) CreateStorageVolume(pool string, name string, args map[string]string) error {
	// Parse the input
	volName, volType := parseVolume("custom", name)

	var volumePut api.StorageVolumePut

	// Create the storage volume entry
	vol := api.StorageVolumesPost{
		Name:             volName,
		Type:             volType,
		ContentType:      "filesystem",
		StorageVolumePut: volumePut,
	}

	if volumePut.Config == nil {
		vol.Config = map[string]string{}
	}

	for k, v := range args {
		vol.Config[k] = v
	}

	err := i.client.CreateStoragePoolVolume(pool, vol)
	if err != nil {
		return err
	}

	return nil
}

func (i *IncusClient) AttachStorageVolume(pool string, name string, service string, mountpoint string) error {
	instance, etag, err := i.client.GetInstance(service)
	if err != nil {
		return err
	}

	volName, volType := parseVolume("custom", name)
	if volType != "custom" {
		return fmt.Errorf("Only \"custom\" volumes can be attached to instances")
	}

	// Prepare the instance's device entry
	dev := map[string]string{
		"type":   "disk",
		"pool":   pool,
		"source": volName,
		"path":   mountpoint,
	}

	// Check if device exists
	_, ok := instance.Devices[name]
	if ok {
		return fmt.Errorf("Device already exists: %s", name)
	}
	instance.Devices[name] = dev

	op, err := i.client.UpdateInstance(service, instance.Writable(), etag)
	if err != nil {
		return err
	}

	return op.Wait()
}

func (i *IncusClient) ShowStorageVolume(pool string, name string) error {
	// Parse the input
	volName, volType := parseVolume("custom", name)

	// Get the storage volume entry
	vol, _, err := i.client.GetStoragePoolVolume(pool, volType, volName)
	if err != nil {
		// Give more context on missing volumes.
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			if volType == "custom" {
				return fmt.Errorf("Storage pool volume \"%s/%s\" not found. Try virtual-machine or container for type", volType, volName)
			}

			return fmt.Errorf("Storage pool volume \"%s/%s\" not found", volType, volName)
		}

		return err
	}

	sort.Strings(vol.UsedBy)

	data, err := yaml.Marshal(&vol)
	if err != nil {
		return err
	}

	fmt.Printf("%s", data)

	return nil
}

func (i *IncusClient) DeleteStoragePoolVolume(pool string, name string) error {
	volName, volType := parseVolume("custom", name)

	if err := i.client.DeleteStoragePoolVolume(pool, volType, volName); err != nil {
		return err
	}
	return nil
}

func parseVolume(defaultType string, name string) (string, string) {
	parsedName := strings.SplitN(name, "/", 2)
	if len(parsedName) == 1 {
		return parsedName[0], defaultType
	} else if len(parsedName) == 2 && !slices.Contains([]string{"custom", "image", "container", "virtual-machine"}, parsedName[0]) {
		return name, defaultType
	}

	return parsedName[1], parsedName[0]
}
