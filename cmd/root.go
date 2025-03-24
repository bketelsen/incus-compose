/*
Copyright © 2024 Brian Ketelsen <bketelsen@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/bketelsen/incus-compose/pkg/application"
	"github.com/bketelsen/incus-compose/pkg/compose"
	"github.com/spf13/viper"

	dockercompose "github.com/compose-spec/compose-go/v2/types"
	"github.com/dominikbraun/graph"
	config "github.com/lxc/incus/v6/shared/cliconfig"
	"github.com/lxc/incus/v6/shared/util"

	"github.com/bketelsen/toolbox/cobra"
	goversion "github.com/bketelsen/toolbox/go-version"
)

var debug bool
var conf *config.Config
var confPath string
var forceLocal bool

// var app application.Compose
var timeout int
var dryRun bool
var cwd string
var project *dockercompose.Project
var app *application.Compose

var appname = "incus-compose"
var (
	version   = ""
	commit    = ""
	treeState = ""
	date      = ""
	builtBy   = ""
)

var bversion = buildVersion(version, commit, date, builtBy, treeState)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "incus-compose",

	InitConfig: func() *viper.Viper {
		config := viper.New()
		config.SetEnvPrefix(appname)
		config.AutomaticEnv()
		config.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", ""))
		config.SetConfigType("yaml")
		config.SetConfigName("incus-compose.yaml")          // name of config file
		config.AddConfigPath("$HOME/.config/incus-compose") // call multiple times to add many search paths
		config.AddConfigPath(".")                           // optionally look for config in the working directory
		if err := config.ReadInConfig(); err == nil {
			slog.Info("Using config file:", slog.String("file", config.ConfigFileUsed()))
		} else {
			slog.Info("No config file found, using defaults")
		}
		return config
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		// set the slog default logger to the cobra logger
		slog.SetDefault(cmd.Logger)
		// set log level based on the --verbose flag
		if cmd.GlobalConfig().GetBool("verbose") {
			debug = true
			cmd.SetLogLevel(slog.LevelDebug)
			cmd.Logger.Debug("Debug logging enabled")
		}
		// skip all the rest for documentation generation
		if cmd.Name() == "gendocs" {
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
			forceLocal = true
		}

		confPath = os.ExpandEnv(path.Join(configDir, "config.yml"))

		// Load the configuration
		if forceLocal {
			conf = config.NewConfig("", true)
		} else if util.PathExists(confPath) {
			conf, err = config.LoadConfig(confPath)
			if err != nil {
				return err
			}
		} else {
			conf = config.NewConfig(filepath.Dir(confPath), true)
		}

		// Set cache directory in config.
		conf.CacheDir = cachePath

		conf.ProjectOverride = os.Getenv("INCUS_PROJECT")

		loader := configureLoader(cmd)
		project, err = loader.LoadProject(context.Background())
		if err != nil {
			return err
		}
		app, err = application.BuildDirect(project, conf)
		if err != nil {
			return err
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
		app.Dag = g
		return nil
	},
	Version: bversion.String(),
	Short:   "Define and run multi-instance applications with Incus",
	Long:    `Define and run multi-instance applications with Incus`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	rootCmd.PersistentFlags().StringVar(&cwd, "cwd", "", "change working directory")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print commands that would be executed without running them")
	rootCmd.PersistentFlags().BoolVarP(&debug, "verbose", "d", false, "verbose logging")
}

func configureLoader(cmd *cobra.Command) compose.Loader {

	o := compose.LoaderOptions{}

	// o.ConfigPaths, err = f.GetStringArray("file")
	// if err != nil {
	// 	panic(err)
	// }
	o.WorkingDir = cmd.GlobalConfig().GetString("cwd")

	// o.ProjectName, err = f.GetString("project-name")
	// if err != nil {
	// 	panic(err)
	// }
	return compose.NewLoaderWithOptions(o)
}

// https://www.asciiart.eu/text-to-ascii-art to make your own
// just make sure the font doesn't have backticks in the letters or
// it will break the string quoting
var asciiName = `
 ██████╗ ██████╗ ███╗   ███╗██████╗  ██████╗ ███████╗███████╗
██╔════╝██╔═══██╗████╗ ████║██╔══██╗██╔═══██╗██╔════╝██╔════╝
██║     ██║   ██║██╔████╔██║██████╔╝██║   ██║███████╗█████╗  
██║     ██║   ██║██║╚██╔╝██║██╔═══╝ ██║   ██║╚════██║██╔══╝  
╚██████╗╚██████╔╝██║ ╚═╝ ██║██║     ╚██████╔╝███████║███████╗
 ╚═════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝      ╚═════╝ ╚══════╝╚══════╝
`

// buildVersion builds the version info for the application
func buildVersion(version, commit, date, builtBy, treeState string) goversion.Info {
	return goversion.GetVersionInfo(
		goversion.WithAppDetails(appname, "An application that does cool things.", "https://github.com/bketelsen/incus-compose"),
		goversion.WithASCIIName(asciiName),
		func(i *goversion.Info) {
			if commit != "" {
				i.GitCommit = commit
			}
			if treeState != "" {
				i.GitTreeState = treeState
			}
			if date != "" {
				i.BuildDate = date
			}
			if version != "" {
				i.GitVersion = version
			}
			if builtBy != "" {
				i.BuiltBy = builtBy
			}

		},
	)
}
