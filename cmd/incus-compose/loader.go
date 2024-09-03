package main

import (
	"github.com/bketelsen/incus-compose/pkg/compose"

	"github.com/spf13/cobra"
)

func configureLoader(cmd *cobra.Command) compose.Loader {
	f := cmd.Flags()
	o := compose.LoaderOptions{}
	var err error

	o.WorkingDir, err = f.GetString("cwd")
	if err != nil {
		panic(err)
	}

	return compose.NewLoaderWithOptions(o)
}
