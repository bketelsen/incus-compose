package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cli "github.com/bketelsen/incus-compose/internal/cmd"
	"github.com/bketelsen/incus-compose/internal/i18n"
	"github.com/bketelsen/incus-compose/internal/instance"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"github.com/lxc/incus/v6/shared/termios"
)

type volumeColumn struct {
	Name       string
	Data       func(api.StorageVolume, api.StorageVolumeState) string
	NeedsState bool
}

type cmdStorageVolume struct {
	global                *cmdGlobal
	storage               *cmdStorage
	flagDestinationTarget string
}

func parseVolume(defaultType string, name string) (string, string) {
	fields := strings.SplitN(name, "/", 2)
	if len(fields) == 1 {
		return fields[0], defaultType
	} else if len(fields) == 2 && !slices.Contains([]string{"custom", "image", "container", "virtual-machine"}, fields[0]) {
		return name, defaultType
	}

	return fields[1], fields[0]
}

func (c *cmdStorageVolume) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("volume")
	cmd.Short = i18n.G("Manage storage volumes")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Manage storage volumes

Unless specified through a prefix, all volume operations affect "custom" (user created) volumes.`))

	// Attach
	storageVolumeAttachCmd := cmdStorageVolumeAttach{global: c.global, storage: c.storage, storageVolume: c}
	cmd.AddCommand(storageVolumeAttachCmd.Command())

	// Attach profile
	storageVolumeAttachProfileCmd := cmdStorageVolumeAttachProfile{global: c.global, storage: c.storage, storageVolume: c}
	cmd.AddCommand(storageVolumeAttachProfileCmd.Command())

	// Create
	storageVolumeCreateCmd := cmdStorageVolumeCreate{global: c.global, storage: c.storage, storageVolume: c}
	cmd.AddCommand(storageVolumeCreateCmd.Command())

	// Set
	storageVolumeSetCmd := cmdStorageVolumeSet{global: c.global, storage: c.storage, storageVolume: c}
	cmd.AddCommand(storageVolumeSetCmd.Command())

	// Show
	storageVolumeShowCmd := cmdStorageVolumeShow{global: c.global, storage: c.storage, storageVolume: c}
	cmd.AddCommand(storageVolumeShowCmd.Command())

	// Snapshot
	storageVolumeSnapshotCmd := cmdStorageVolumeSnapshot{global: c.global, storage: c.storage, storageVolume: c}
	cmd.AddCommand(storageVolumeSnapshotCmd.Command())

	// Workaround for subcommand usage errors. See: https://github.com/spf13/cobra/issues/706
	cmd.Args = cobra.NoArgs
	cmd.Run = func(cmd *cobra.Command, args []string) { _ = cmd.Usage() }
	return cmd
}

func parseVolumeWithPool(name string) (string, string) {
	fields := strings.SplitN(name, "/", 2)
	if len(fields) == 1 {
		return fields[0], ""
	}

	return fields[1], fields[0]
}

// Attach.
type cmdStorageVolumeAttach struct {
	global        *cmdGlobal
	storage       *cmdStorage
	storageVolume *cmdStorageVolume
}

func (c *cmdStorageVolumeAttach) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("attach", i18n.G("[<remote>:]<pool> <volume> <instance> [<device name>] [<path>]"))
	cmd.Short = i18n.G("Attach new custom storage volumes to instances")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Attach new custom storage volumes to instances`))

	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		if len(args) == 1 {
			return c.global.cmpStoragePoolVolumes(args[0])
		}

		if len(args) == 2 {
			return c.global.cmpInstanceNamesFromRemote(args[0])
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeAttach) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 3, 5)
	if exit {
		return err
	}

	// Parse remote
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]

	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	// Attach the volume
	devPath := ""
	devName := ""
	if len(args) == 3 {
		devName = args[1]
	} else if len(args) == 4 {
		// Only the path has been given to us.
		devPath = args[3]
		devName = args[1]
	} else if len(args) == 5 {
		// Path and device name have been given to us.
		devName = args[3]
		devPath = args[4]
	}

	volName, volType := parseVolume("custom", args[1])
	if volType != "custom" {
		return fmt.Errorf(i18n.G("Only \"custom\" volumes can be attached to instances"))
	}

	// Prepare the instance's device entry
	device := map[string]string{
		"type":   "disk",
		"pool":   resource.name,
		"source": volName,
		"path":   devPath,
	}

	// Add the device to the instance
	err = instanceDeviceAdd(resource.server, args[2], devName, device)
	if err != nil {
		return err
	}

	return nil
}

// Attach profile.
type cmdStorageVolumeAttachProfile struct {
	global        *cmdGlobal
	storage       *cmdStorage
	storageVolume *cmdStorageVolume
}

func (c *cmdStorageVolumeAttachProfile) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("attach-profile", i18n.G("[<remote:>]<pool> <volume> <profile> [<device name>] [<path>]"))
	cmd.Short = i18n.G("Attach new custom storage volumes to profiles")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Attach new custom storage volumes to profiles`))

	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		if len(args) == 1 {
			return c.global.cmpStoragePoolVolumes(args[0])
		}

		if len(args) == 2 {
			return c.global.cmpProfileNamesFromRemote(args[0])
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeAttachProfile) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 3, 5)
	if exit {
		return err
	}

	// Parse remote
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]

	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	// Attach the volume
	devPath := ""
	devName := ""
	if len(args) == 3 {
		devName = args[1]
	} else if len(args) == 4 {
		// Only the path has been given to us.
		devPath = args[3]
		devName = args[1]
	} else if len(args) == 5 {
		// Path and device name have been given to us.
		devName = args[3]
		devPath = args[4]
	}

	volName, volType := parseVolume("custom", args[1])
	if volType != "custom" {
		return fmt.Errorf(i18n.G("Only \"custom\" volumes can be attached to instances"))
	}

	// Check if the requested storage volume actually exists
	vol, _, err := resource.server.GetStoragePoolVolume(resource.name, volType, volName)
	if err != nil {
		return err
	}

	// Prepare the instance's device entry
	device := map[string]string{
		"type":   "disk",
		"pool":   resource.name,
		"source": vol.Name,
	}

	// Ignore path for block volumes
	if vol.ContentType != "block" {
		device["path"] = devPath
	}

	// Add the device to the instance
	err = profileDeviceAdd(resource.server, args[2], devName, device)
	if err != nil {
		return err
	}

	return nil
}

// Create.
type cmdStorageVolumeCreate struct {
	global          *cmdGlobal
	storage         *cmdStorage
	storageVolume   *cmdStorageVolume
	flagContentType string
}

func (c *cmdStorageVolumeCreate) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("create", i18n.G("[<remote>:]<pool> <volume> [key=value...]"))
	cmd.Short = i18n.G("Create new custom storage volumes")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Create new custom storage volumes`))
	cmd.Example = cli.FormatSection("", i18n.G(`incus storage volume create default foo
    Create custom storage volume "foo" in pool "default"

incus storage volume create default foo < config.yaml
    Create custom storage volume "foo" in pool "default" with configuration from config.yaml`))

	cmd.Flags().StringVar(&c.storage.flagTarget, "target", "", i18n.G("Cluster member name")+"``")
	cmd.Flags().StringVar(&c.flagContentType, "type", "filesystem", i18n.G("Content type, block or filesystem")+"``")
	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeCreate) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 2, -1)
	if exit {
		return err
	}

	// Parse remote
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]

	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	var volumePut api.StorageVolumePut
	if !termios.IsTerminal(getStdinFd()) {
		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		err = yaml.UnmarshalStrict(contents, &volumePut)
		if err != nil {
			return err
		}
	}

	// Parse the input
	volName, volType := parseVolume("custom", args[1])

	// Create the storage volume entry
	vol := api.StorageVolumesPost{
		Name:             volName,
		Type:             volType,
		ContentType:      c.flagContentType,
		StorageVolumePut: volumePut,
	}

	if volumePut.Config == nil {
		vol.Config = map[string]string{}
	}

	for i := 2; i < len(args); i++ {
		entry := strings.SplitN(args[i], "=", 2)
		if len(entry) < 2 {
			return fmt.Errorf(i18n.G("Bad key=value pair: %s"), entry)
		}

		vol.Config[entry[0]] = entry[1]
	}

	// If a target was specified, create the volume on the given member.
	if c.storage.flagTarget != "" {
		client = client.UseTarget(c.storage.flagTarget)
	}

	err = client.CreateStoragePoolVolume(resource.name, vol)
	if err != nil {
		return err
	}

	if !c.global.flagQuiet {
		fmt.Printf(i18n.G("Storage volume %s created")+"\n", args[1])
	}

	return nil
}

// Delete.
type cmdStorageVolumeDelete struct {
	global *cmdGlobal
}

func (c *cmdStorageVolumeDelete) Run(pool, volume, target string) error {

	// Parse remote
	resources, err := c.global.ParseServers(pool)
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	// Parse the input
	volName, volType := parseVolume("custom", volume)

	// If a target was specified, delete the volume on the given member.
	if target != "" {
		client = client.UseTarget(target)
	}

	// Delete the volume
	err = client.DeleteStoragePoolVolume(resource.name, volType, volName)
	if err != nil {
		return err
	}

	if !c.global.flagQuiet {
		fmt.Printf(i18n.G("Storage volume %s deleted")+"\n", volume)
	}

	return nil
}

// Set.
type cmdStorageVolumeSet struct {
	global        *cmdGlobal
	storage       *cmdStorage
	storageVolume *cmdStorageVolume

	flagIsProperty bool
}

func (c *cmdStorageVolumeSet) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("set", i18n.G("[<remote>:]<pool> [<type>/]<volume> <key>=<value>..."))
	cmd.Short = i18n.G("Set storage volume configuration keys")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Set storage volume configuration keys

For backward compatibility, a single configuration key may still be set with:
    incus storage volume set [<remote>:]<pool> [<type>/]<volume> <key> <value>

If the type is not specified, Incus assumes the type is "custom".
Supported values for type are "custom", "container" and "virtual-machine".`))
	cmd.Example = cli.FormatSection("", i18n.G(
		`incus storage volume set default data size=1GiB
    Sets the size of a custom volume "data" in pool "default" to 1 GiB

incus storage volume set default virtual-machine/data snapshots.expiry=7d
    Sets the snapshot expiration period for a virtual machine "data" in pool "default" to seven days`))

	cmd.Flags().StringVar(&c.storage.flagTarget, "target", "", i18n.G("Cluster member name")+"``")
	cmd.Flags().BoolVarP(&c.flagIsProperty, "property", "p", false, i18n.G("Set the key as a storage volume property"))
	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		if len(args) == 1 {
			return c.global.cmpStoragePoolVolumes(args[0])
		}

		// TODO all volume config keys

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeSet) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 3, -1)
	if exit {
		return err
	}

	// Parse remote.
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	// Get the values.
	keys, err := getConfig(args[2:]...)
	if err != nil {
		return err
	}

	// Parse the input.
	volName, volType := parseVolume("custom", args[1])

	isSnapshot := false
	fields := strings.Split(volName, "/")
	if len(fields) > 2 {
		return fmt.Errorf(i18n.G("Invalid snapshot name"))
	} else if len(fields) > 1 {
		isSnapshot = true
	}

	if isSnapshot {
		if c.flagIsProperty {
			snapVol, etag, err := client.GetStoragePoolVolumeSnapshot(resource.name, volType, fields[0], fields[1])
			if err != nil {
				return err
			}

			writable := snapVol.Writable()
			if cmd.Name() == "unset" {
				for k := range keys {
					err := unsetFieldByJsonTag(&writable, k)
					if err != nil {
						return fmt.Errorf(i18n.G("Error unsetting property: %v"), err)
					}
				}
			} else {
				err := unpackKVToWritable(&writable, keys)
				if err != nil {
					return fmt.Errorf(i18n.G("Error setting properties: %v"), err)
				}
			}

			err = client.UpdateStoragePoolVolumeSnapshot(resource.name, volType, fields[0], fields[1], writable, etag)
			if err != nil {
				return err
			}

			return nil
		}

		return fmt.Errorf(i18n.G("Snapshots are read-only and can't have their configuration changed"))
	}

	// If a target was specified, create the volume on the given member.
	if c.storage.flagTarget != "" {
		client = client.UseTarget(c.storage.flagTarget)
	}

	// Get the storage volume entry.
	vol, etag, err := client.GetStoragePoolVolume(resource.name, volType, volName)
	if err != nil {
		// Give more context on missing volumes.
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return fmt.Errorf("Storage pool volume \"%s/%s\" not found", volType, volName)
		}

		return err
	}

	writable := vol.Writable()
	if c.flagIsProperty {
		if cmd.Name() == "unset" {
			for k := range keys {
				err := unsetFieldByJsonTag(&writable, k)
				if err != nil {
					return fmt.Errorf(i18n.G("Error unsetting property: %v"), err)
				}
			}
		} else {
			err := unpackKVToWritable(&writable, keys)
			if err != nil {
				return fmt.Errorf(i18n.G("Error setting properties: %v"), err)
			}
		}
	} else {
		// Update the volume config keys.
		for k, v := range keys {
			writable.Config[k] = v
		}
	}

	err = client.UpdateStoragePoolVolume(resource.name, vol.Type, vol.Name, writable, etag)
	if err != nil {
		return err
	}

	return nil
}

// Show.
type cmdStorageVolumeShow struct {
	global        *cmdGlobal
	storage       *cmdStorage
	storageVolume *cmdStorageVolume
}

func (c *cmdStorageVolumeShow) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("show", i18n.G("[<remote>:]<pool> [<type>/]<volume>"))
	cmd.Short = i18n.G("Show storage volume configurations")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Show storage volume configurations

If the type is not specified, Incus assumes the type is "custom".
Supported values for type are "custom", "container" and "virtual-machine".

For snapshots, add the snapshot name (only if type is one of custom, container or virtual-machine).`))
	cmd.Example = cli.FormatSection("", i18n.G(`incus storage volume show default foo
    Will show the properties of custom volume "foo" in pool "default"

incus storage volume show default virtual-machine/v1
    Will show the properties of the virtual-machine volume "v1" in pool "default"

incus storage volume show default container/c1
    Will show the properties of the container volume "c1" in pool "default"`))

	cmd.Flags().StringVar(&c.storage.flagTarget, "target", "", i18n.G("Cluster member name")+"``")
	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		if len(args) == 1 {
			return c.global.cmpStoragePoolVolumes(args[0])
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeShow) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 2, 2)
	if exit {
		return err
	}

	// Parse remote
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]

	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	// Parse the input
	volName, volType := parseVolume("custom", args[1])

	// If a target member was specified, get the volume with the matching
	// name on that member, if any.
	if c.storage.flagTarget != "" {
		client = client.UseTarget(c.storage.flagTarget)
	}

	// Get the storage volume entry
	vol, _, err := client.GetStoragePoolVolume(resource.name, volType, volName)
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

// Snapshot.
type cmdStorageVolumeSnapshot struct {
	global        *cmdGlobal
	storage       *cmdStorage
	storageVolume *cmdStorageVolume
}

func (c *cmdStorageVolumeSnapshot) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("snapshot")
	cmd.Short = i18n.G("Manage storage volume snapshots")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Manage storage volume snapshots`))

	// Create
	storageVolumeSnapshotCreateCmd := cmdStorageVolumeSnapshotCreate{global: c.global, storage: c.storage, storageVolume: c.storageVolume, storageVolumeSnapshot: c}
	cmd.AddCommand(storageVolumeSnapshotCreateCmd.Command())

	// List
	storageVolumeSnapshotListCmd := cmdStorageVolumeSnapshotList{global: c.global, storage: c.storage, storageVolume: c.storageVolume, storageVolumeSnapshot: c}
	cmd.AddCommand(storageVolumeSnapshotListCmd.Command())

	// Restore
	storageVolumeSnapshotShowCmd := cmdStorageVolumeSnapshotShow{global: c.global, storage: c.storage, storageVolume: c.storageVolume, storageVolumeSnapshot: c}
	cmd.AddCommand(storageVolumeSnapshotShowCmd.Command())

	// Workaround for subcommand usage errors. See: https://github.com/spf13/cobra/issues/706
	cmd.Args = cobra.NoArgs
	cmd.Run = func(cmd *cobra.Command, args []string) { _ = cmd.Usage() }

	return cmd
}

// Snapshot create.
type cmdStorageVolumeSnapshotCreate struct {
	global                *cmdGlobal
	storage               *cmdStorage
	storageVolume         *cmdStorageVolume
	storageVolumeSnapshot *cmdStorageVolumeSnapshot

	flagNoExpiry bool
	flagReuse    bool
}

func (c *cmdStorageVolumeSnapshotCreate) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("create", i18n.G("[<remote>:]<pool> <volume> [<snapshot>]"))
	cmd.Short = i18n.G("Snapshot storage volumes")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Snapshot storage volumes`))
	cmd.Example = cli.FormatSection("", i18n.G(`incus storage volume snapshot create default foo snap0
    Create a snapshot of "foo" in pool "default" called "snap0"

incus storage volume snapshot create default vol1 snap0 < config.yaml
    Create a snapshot of "foo" in pool "default" called "snap0" with the configuration from "config.yaml"`))

	cmd.Flags().BoolVar(&c.flagNoExpiry, "no-expiry", false, i18n.G("Ignore any configured auto-expiry for the storage volume"))
	cmd.Flags().BoolVar(&c.flagReuse, "reuse", false, i18n.G("If the snapshot name already exists, delete and create a new one"))
	cmd.Flags().StringVar(&c.storage.flagTarget, "target", "", i18n.G("Cluster member name")+"``")
	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		if len(args) == 1 {
			return c.global.cmpStoragePoolVolumes(args[0])
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeSnapshotCreate) Run(cmd *cobra.Command, args []string) error {
	var stdinData api.StorageVolumeSnapshotPut

	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 2, 3)
	if exit {
		return err
	}

	// If stdin isn't a terminal, read text from it
	if !termios.IsTerminal(getStdinFd()) {
		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(contents, &stdinData)
		if err != nil {
			return err
		}
	}

	// Parse remote
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	// Use the provided target.
	if c.storage.flagTarget != "" {
		client = client.UseTarget(c.storage.flagTarget)
	}

	// Parse the input
	volName, volType := parseVolume("custom", args[1])
	if volType != "custom" {
		return fmt.Errorf(i18n.G("Only \"custom\" volumes can be snapshotted"))
	}

	// Check if the requested storage volume actually exists
	_, _, err = client.GetStoragePoolVolume(resource.name, volType, volName)
	if err != nil {
		return err
	}

	var snapname string
	if len(args) < 3 {
		snapname = ""
	} else {
		snapname = args[2]
	}

	req := api.StorageVolumeSnapshotsPost{
		Name: snapname,
	}

	if c.flagNoExpiry {
		req.ExpiresAt = &time.Time{}
	} else if stdinData.ExpiresAt != nil && !stdinData.ExpiresAt.IsZero() {
		req.ExpiresAt = stdinData.ExpiresAt
	}

	if c.flagReuse && snapname != "" {
		snap, _, _ := client.GetStoragePoolVolumeSnapshot(resource.name, volType, volName, snapname)
		if snap != nil {
			op, err := client.DeleteStoragePoolVolumeSnapshot(resource.name, volType, volName, snapname)
			if err != nil {
				return err
			}

			err = op.Wait()
			if err != nil {
				return err
			}
		}
	}

	op, err := client.CreateStoragePoolVolumeSnapshot(resource.name, volType, volName, req)
	if err != nil {
		return err
	}

	return op.Wait()
}

// Snapshot list.
type cmdStorageVolumeSnapshotList struct {
	global                *cmdGlobal
	storage               *cmdStorage
	storageVolume         *cmdStorageVolume
	storageVolumeSnapshot *cmdStorageVolumeSnapshot

	flagFormat      string
	flagColumns     string
	flagAllProjects bool

	defaultColumns string
}

func (c *cmdStorageVolumeSnapshotList) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("list", i18n.G("[<remote>:]<pool> <volume>"))
	cmd.Short = i18n.G("List storage volume snapshots")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`List storage volume snapshots`))

	c.defaultColumns = "etndcuL"
	cmd.Flags().StringVarP(&c.flagColumns, "columns", "c", c.defaultColumns, i18n.G("Columns")+"``")
	cmd.Flags().BoolVar(&c.flagAllProjects, "all-projects", false, i18n.G("All projects")+"``")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`List storage volume snapshots

	The -c option takes a (optionally comma-separated) list of arguments
	that control which image attributes to output when displaying in table
	or csv format.

	Column shorthand chars:
		c - Content type (filesystem or block)
		d - Description
		e - Project name
		L - Location of the instance (e.g. its cluster member)
		n - Name
		t - Type of volume (custom, image, container or virtual-machine)
		u - Number of references (used by)`))
	cmd.Flags().StringVarP(&c.flagFormat, "format", "f", "table", i18n.G("Format (csv|json|table|yaml|compact)")+"``")

	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		if len(args) == 1 {
			return c.global.cmpStoragePoolVolumes(args[0])
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeSnapshotList) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 2, -1)
	if exit {
		return err
	}

	// Parse remote
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]

	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	// Parse the input
	volName, volType := parseVolume("custom", args[1])

	// Check if the requested storage volume actually exists
	_, _, err = resource.server.GetStoragePoolVolume(resource.name, volType, volName)
	if err != nil {
		return err
	}

	return c.listSnapshots(resource.server, resource.name, volType, volName)
}

func (c *cmdStorageVolumeSnapshotList) listSnapshots(d incus.InstanceServer, poolName string, volumeType string, volumeName string) error {
	snapshots, err := d.GetStoragePoolVolumeSnapshots(poolName, volumeType, volumeName)
	if err != nil {
		return err
	}

	// List snapshots
	snapData := [][]string{}

	for _, snap := range snapshots {
		var row []string

		fields := strings.Split(snap.Name, instance.SnapshotDelimiter)
		row = append(row, fields[len(fields)-1])

		if !snap.CreatedAt.IsZero() {
			row = append(row, snap.CreatedAt.Local().Format(dateLayout))
		} else {
			row = append(row, " ")
		}

		if snap.ExpiresAt != nil && !snap.ExpiresAt.IsZero() {
			row = append(row, snap.ExpiresAt.Local().Format(dateLayout))
		} else {
			row = append(row, " ")
		}

		snapData = append(snapData, row)
	}

	snapHeader := []string{
		i18n.G("Name"),
		i18n.G("Taken at"),
		i18n.G("Expires at"),
	}

	_ = cli.RenderTable(cli.TableFormatTable, snapHeader, snapData, snapshots)

	return nil
}

// Snapshot show.
type cmdStorageVolumeSnapshotShow struct {
	global                *cmdGlobal
	storage               *cmdStorage
	storageVolume         *cmdStorageVolume
	storageVolumeSnapshot *cmdStorageVolumeSnapshot
}

func (c *cmdStorageVolumeSnapshotShow) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = usage("show", i18n.G("[<remote>:]<pool> <volume>/<snapshot>"))
	cmd.Short = i18n.G("Show storage volume snapshot configurations")
	cmd.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Show storage volume snapshhot configurations`))
	cmd.Flags().StringVar(&c.storage.flagTarget, "target", "", i18n.G("Cluster member name")+"``")

	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return c.global.cmpStoragePools(toComplete)
		}

		if len(args) == 1 {
			return c.global.cmpStoragePoolVolumes(args[0])
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func (c *cmdStorageVolumeSnapshotShow) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 2, 2)
	if exit {
		return err
	}

	// Parse remote
	resources, err := c.global.ParseServers(args[0])
	if err != nil {
		return err
	}

	resource := resources[0]

	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	// Parse the input
	volName, volType := parseVolume("custom", args[1])

	fields := strings.Split(volName, "/")
	if len(fields) != 2 {
		return fmt.Errorf(i18n.G("Invalid snapshot name"))
	}

	// If a target member was specified, get the volume with the matching
	// name on that member, if any.
	if c.storage.flagTarget != "" {
		client = client.UseTarget(c.storage.flagTarget)
	}

	// Get the storage volume entry
	vol, _, err := client.GetStoragePoolVolumeSnapshot(resource.name, volType, fields[0], fields[1])
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(&vol)
	if err != nil {
		return err
	}

	fmt.Printf("%s", data)

	return nil
}
