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
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bketelsen/incus-compose/pkg/application"
	"github.com/bketelsen/incus-compose/pkg/build"
	"github.com/bketelsen/incus-compose/pkg/compose"
	"gopkg.in/yaml.v3"

	dockercompose "github.com/compose-spec/compose-go/v2/types"
	"github.com/dominikbraun/graph"
	"github.com/lxc/incus/shared/util"
	config "github.com/lxc/incus/v6/shared/cliconfig"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var debug bool
var conf *config.Config
var confPath string
var forceLocal bool

// var app application.Compose
var logLevel = new(slog.LevelVar) // Info by default
var timeout int
var dryRun bool
var cwd string
var project *dockercompose.Project
var app *application.Compose

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "incus-compose",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {

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

		globalPreRunHook(cmd, args)
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
		if debug {
			debugProject()
			fmt.Println()
			debugCompose()
		}

		return nil
	},

	Short:   "Define and run multi-instance applications with Incus",
	Long:    `Define and run multi-instance applications with Incus`,
	Version: build.Version,
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
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cwd, "cwd", "", "change working directory")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print commands that would be executed without running them")
	rootCmd.PersistentFlags().BoolVarP(&debug, "verbose", "d", false, "verbose logging")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

}
func globalPreRunHook(_ *cobra.Command, _ []string) {

	// set up logging
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.Kitchen,
		}),
	))
	if debug {
		logLevel.Set(slog.LevelDebug)
		slog.Debug("Verbose logging")
	} else {
		logLevel.Set(getLogLevelFromEnv())
	}

}

func getLogLevelFromEnv() slog.Level {
	levelStr := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo

	}
}

func configureLoader(cmd *cobra.Command) compose.Loader {
	f := cmd.Flags()
	o := compose.LoaderOptions{}
	var err error

	// o.ConfigPaths, err = f.GetStringArray("file")
	// if err != nil {
	// 	panic(err)
	// }

	o.WorkingDir, err = f.GetString("cwd")
	if err != nil {
		panic(err)
	}

	// o.ProjectName, err = f.GetString("project-name")
	// if err != nil {
	// 	panic(err)
	// }
	return compose.NewLoaderWithOptions(o)
}

func debugCompose() {
	fmt.Println("Compose:")
	fmt.Println(app.String())

}

func debugProject() {
	fmt.Println("Project:")
	bb, _ := yaml.Marshal(project)
	fmt.Println(string(bb))
}
