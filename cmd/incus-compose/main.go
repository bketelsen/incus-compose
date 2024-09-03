package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"slices"

	"github.com/dominikbraun/graph"
	"github.com/spf13/cobra"

	cli "github.com/bketelsen/incus-compose/internal/cmd"
	"github.com/bketelsen/incus-compose/internal/i18n"
	internalUtil "github.com/bketelsen/incus-compose/internal/util"
	"github.com/bketelsen/incus-compose/internal/version"
	"github.com/bketelsen/incus-compose/pkg/application"
	dockercompose "github.com/compose-spec/compose-go/v2/types"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	config "github.com/lxc/incus/v6/shared/cliconfig"
	"github.com/lxc/incus/v6/shared/logger"
	"github.com/lxc/incus/v6/shared/util"
)

type cmdGlobal struct {
	asker cli.Asker

	conf     *config.Config
	confPath string
	cmd      *cobra.Command
	ret      int
	project  *dockercompose.Project
	compose  *application.Compose

	flagForceLocal bool
	flagHelp       bool
	flagHelpAll    bool
	flagLogDebug   bool
	flagLogVerbose bool
	flagProject    string
	flagQuiet      bool
	flagVersion    bool
	flagSubCmds    bool

	cwd string
}

func usageTemplateSubCmds() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}
Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }}  {{.Short}}{{if .HasSubCommands}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
    {{rpad .Name .NamePadding }}  {{.Short}}{{if .HasSubCommands}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
      {{rpad .Name .NamePadding }}  {{.Short}}{{if .HasSubCommands}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
        {{rpad .Name .NamePadding }}  {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
  {{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}

func main() {
	// Process aliases
	err := execIfAliases()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Setup the parser
	app := &cobra.Command{}
	app.Use = "incus-compose"
	app.Short = i18n.G("Define and run multi-instance applications with Incus")
	app.Long = cli.FormatSection(i18n.G("Description"), i18n.G(
		`Command line client for Incus

All of Incus's features can be driven through the various commands below.
For help with any of those, simply call them with --help.

Custom commands can be defined through aliases, use "incus alias" to control those.`))
	app.SilenceUsage = true
	app.SilenceErrors = true
	app.CompletionOptions = cobra.CompletionOptions{HiddenDefaultCmd: true}

	// Global flags
	globalCmd := cmdGlobal{cmd: app, asker: cli.NewAsker(bufio.NewReader(os.Stdin))}

	app.PersistentFlags().BoolVar(&globalCmd.flagVersion, "version", false, i18n.G("Print version number"))
	app.PersistentFlags().BoolVarP(&globalCmd.flagHelp, "help", "h", false, i18n.G("Print help"))
	app.PersistentFlags().BoolVar(&globalCmd.flagForceLocal, "force-local", false, i18n.G("Force using the local unix socket"))
	app.PersistentFlags().StringVar(&globalCmd.flagProject, "project", "", i18n.G("Override the source project")+"``")
	app.PersistentFlags().BoolVar(&globalCmd.flagLogDebug, "debug", false, i18n.G("Show all debug messages"))
	app.PersistentFlags().BoolVarP(&globalCmd.flagLogVerbose, "verbose", "v", false, i18n.G("Show all information messages"))
	app.PersistentFlags().BoolVarP(&globalCmd.flagQuiet, "quiet", "q", false, i18n.G("Don't show progress information"))
	app.PersistentFlags().BoolVar(&globalCmd.flagSubCmds, "sub-commands", false, i18n.G("Use with help or --help to view sub-commands"))
	app.PersistentFlags().StringVar(&globalCmd.cwd, "cwd", "", "change working directory")

	// Wrappers
	app.PersistentPreRunE = globalCmd.PreRun
	app.PersistentPostRunE = globalCmd.PostRun

	// Version handling
	app.SetVersionTemplate("{{.Version}}\n")
	app.Version = version.Version

	// console sub-command
	consoleCmd := cmdConsole{global: &globalCmd}
	app.AddCommand(consoleCmd.Command())

	// restart sub-command
	restartCmd := cmdRestart{global: &globalCmd}
	app.AddCommand(restartCmd.Command())

	// resume sub-command
	resumeCmd := cmdResume{global: &globalCmd}
	app.AddCommand(resumeCmd.Command())

	// pause sub-command
	pauseCmd := cmdPause{global: &globalCmd}
	app.AddCommand(pauseCmd.Command())

	// start sub-command
	startCmd := cmdStart{global: &globalCmd}
	app.AddCommand(startCmd.Command())

	// stop sub-command
	stopCmd := cmdStop{global: &globalCmd}
	app.AddCommand(stopCmd.Command())

	// down sub-command
	downCmd := cmdDown{global: &globalCmd}
	app.AddCommand(downCmd.Command())

	// up sub-command
	upCmd := cmdUp{global: &globalCmd}
	app.AddCommand(upCmd.Command())

	// Get help command
	app.InitDefaultHelpCmd()
	var help *cobra.Command
	for _, cmd := range app.Commands() {
		if cmd.Name() == "help" {
			help = cmd
			break
		}
	}

	// Help flags
	app.Flags().BoolVar(&globalCmd.flagHelpAll, "all", false, i18n.G("Show less common commands"))
	help.Flags().BoolVar(&globalCmd.flagHelpAll, "all", false, i18n.G("Show less common commands"))

	// Deal with --all flag and --sub-commands flag
	err = app.ParseFlags(os.Args[1:])
	if err == nil {
		if globalCmd.flagHelpAll {
			// Show all commands
			for _, cmd := range app.Commands() {
				if cmd.Name() == "completion" {
					continue
				}

				cmd.Hidden = false
			}
		}

		if globalCmd.flagSubCmds {
			app.SetUsageTemplate(usageTemplateSubCmds())
		}
	}

	// Run the main command and handle errors
	err = app.Execute()
	if err != nil {
		// Handle non-Linux systems
		if err == config.ErrNotLinux {
			fmt.Fprintf(os.Stderr, i18n.G(`This client hasn't been configured to use a remote server yet.
As your platform can't run native Linux instances, you must connect to a remote server.

If you already added a remote server, make it the default with "incus remote switch NAME".`)+"\n")
			os.Exit(1)
		}

		// Default error handling
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// If custom exit status not set, use default error status.
		if globalCmd.ret == 0 {
			globalCmd.ret = 1
		}
	}

	if globalCmd.ret != 0 {
		os.Exit(globalCmd.ret)
	}
}

func (c *cmdGlobal) PreRun(cmd *cobra.Command, args []string) error {
	var err error

	// If calling the help, skip pre-run
	if cmd.Name() == "help" {
		return nil
	}

	// Figure out the config directory and config path
	var configDir string
	if os.Getenv("INCUS_CONF") != "" {
		configDir = os.Getenv("INCUS_CONF")
	} else if os.Getenv("HOME") != "" && util.PathExists(os.Getenv("HOME")) {
		configDir = path.Join(os.Getenv("HOME"), ".config", "incus")
	} else {
		user, err := user.Current()
		if err != nil {
			return err
		}

		if util.PathExists(user.HomeDir) {
			configDir = path.Join(user.HomeDir, ".config", "incus")
		}
	}

	// Figure out a potential cache path.
	var cachePath string
	if os.Getenv("INCUS_CACHE") != "" {
		cachePath = os.Getenv("INCUS_CACHE")
	} else if os.Getenv("HOME") != "" && util.PathExists(os.Getenv("HOME")) {
		cachePath = path.Join(os.Getenv("HOME"), ".cache", "incus")
	} else {
		currentUser, err := user.Current()
		if err != nil {
			return err
		}

		if util.PathExists(currentUser.HomeDir) {
			cachePath = path.Join(currentUser.HomeDir, ".cache", "incus")
		}
	}

	if cachePath != "" {
		err := os.MkdirAll(cachePath, 0700)
		if err != nil && !os.IsExist(err) {
			cachePath = ""
		}
	}

	// If no homedir could be found, treat as if --force-local was passed.
	if configDir == "" {
		c.flagForceLocal = true
	}

	c.confPath = os.ExpandEnv(path.Join(configDir, "config.yml"))

	// Load the configuration
	if c.flagForceLocal {
		c.conf = config.NewConfig("", true)
	} else if util.PathExists(c.confPath) {
		c.conf, err = config.LoadConfig(c.confPath)
		if err != nil {
			return err
		}
	} else {
		c.conf = config.NewConfig(filepath.Dir(c.confPath), true)
	}

	// Set cache directory in config.
	c.conf.CacheDir = cachePath

	// Override the project
	if c.flagProject != "" {
		c.conf.ProjectOverride = c.flagProject
	} else {
		c.conf.ProjectOverride = os.Getenv("INCUS_PROJECT")
	}

	// Setup password helper
	c.conf.PromptPassword = func(filename string) (string, error) {
		return cli.AskPasswordOnce(fmt.Sprintf(i18n.G("Password for %s: "), filename)), nil
	}

	// If the user is running a command that may attempt to connect to the local daemon
	// and this is the first time the client has been run by the user, then check to see
	// if the server has been properly configured.  Don't display the message if the var path
	// does not exist (server missing), as the user may be targeting a remote daemon.
	if !c.flagForceLocal && !util.PathExists(c.confPath) {
		// Create the config dir so that we don't get in here again for this user.
		err = os.MkdirAll(c.conf.ConfigDir, 0750)
		if err != nil {
			return err
		}

		// Handle local servers.
		if util.PathExists(internalUtil.VarPath("")) {
			// Attempt to connect to the local server
			runInit := true
			d, err := incus.ConnectIncusUnix("", nil)
			if err == nil {
				// Check if server is initialized.
				info, _, err := d.GetServer()
				if err == nil && info.Environment.Storage != "" {
					runInit = false
				}

				// Detect usable project.
				names, err := d.GetProjectNames()
				if err == nil {
					if len(names) == 1 && names[0] != api.ProjectDefaultName {
						remote := c.conf.Remotes["local"]
						remote.Project = names[0]
						c.conf.Remotes["local"] = remote
					}
				}
			}

			flush := false
			if runInit && (cmd.Name() != "init" || cmd.Parent() == nil || cmd.Parent().Name() != "admin") {
				fmt.Fprintf(os.Stderr, i18n.G("If this is your first time running Incus on this machine, you should also run: incus admin init")+"\n")
				flush = true
			}

			if !slices.Contains([]string{"admin", "create", "launch"}, cmd.Name()) && (cmd.Parent() == nil || cmd.Parent().Name() != "admin") {
				fmt.Fprintf(os.Stderr, i18n.G(`To start your first container, try: incus launch images:ubuntu/22.04
Or for a virtual machine: incus launch images:ubuntu/22.04 --vm`)+"\n")
				flush = true
			}

			if flush {
				fmt.Fprintf(os.Stderr, "\n")
			}
		}

		// And save the initial configuration
		err = c.conf.SaveConfig(c.confPath)
		if err != nil {
			return err
		}
	}

	// Set the user agent
	c.conf.UserAgent = version.UserAgent

	// Setup the logger
	err = logger.InitLogger("", "", c.flagLogVerbose, c.flagLogDebug, nil)
	if err != nil {
		return err
	}

	// setup the compose loader
	loader := configureLoader(cmd)
	c.project, err = loader.LoadProject(context.Background())
	if err != nil {
		fmt.Println("loader error", err)
		return err
	}
	c.compose, err = application.BuildDirect(c.project)
	if err != nil {
		return err
	}
	g := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic())
	for name := range c.compose.Services {
		_ = g.AddVertex(name)
	}
	for name := range c.compose.Services {
		for _, dep := range c.compose.Services[name].DependsOn {
			_ = g.AddEdge(name, dep)
		}
	}
	c.compose.Dag = g
	// if debug {
	// 	debugProject()
	// 	fmt.Println()
	// 	debugCompose()
	// }
	c.compose.SetConfig(c.conf)
	fmt.Println("prerun done")
	return nil

}

func (c *cmdGlobal) PostRun(cmd *cobra.Command, args []string) error {
	if c.conf != nil && util.PathExists(c.confPath) {
		// Save OIDC tokens on exit
		c.conf.SaveOIDCTokens()
	}

	return nil
}

type remoteResource struct {
	remote string
	server incus.InstanceServer
	name   string
}

func (c *cmdGlobal) ParseServers(remotes ...string) ([]remoteResource, error) {
	servers := map[string]incus.InstanceServer{}
	resources := []remoteResource{}

	for _, remote := range remotes {
		// Parse the remote
		remoteName, name, err := c.conf.ParseRemote(remote)
		if err != nil {
			return nil, err
		}

		// Setup the struct
		resource := remoteResource{
			remote: remoteName,
			name:   name,
		}

		// Look at our cache
		_, ok := servers[remoteName]
		if ok {
			resource.server = servers[remoteName]
			resources = append(resources, resource)
			continue
		}

		// New connection
		d, err := c.conf.GetInstanceServer(remoteName)
		if err != nil {
			return nil, err
		}

		resource.server = d
		servers[remoteName] = d
		resources = append(resources, resource)
	}

	return resources, nil
}

func (c *cmdGlobal) CheckArgs(cmd *cobra.Command, args []string, minArgs int, maxArgs int) (bool, error) {
	if len(args) < minArgs || (maxArgs != -1 && len(args) > maxArgs) {
		_ = cmd.Help()

		if len(args) == 0 {
			return true, nil
		}

		return true, fmt.Errorf(i18n.G("Invalid number of arguments"))
	}

	return false, nil
}
