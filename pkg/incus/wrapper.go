package incus

import (
	"context"

	execute "github.com/alexellis/go-execute/v2"
)

var command = "incus"

// ExecuteShellStream executes the incus command in shell mode with stdio streaming
func ExecuteShellStreamExitCode(context context.Context, args []string) (string, int, error) {
	cmd := execute.ExecTask{
		Command:     command,
		Args:        args,
		StreamStdio: true,
		Shell:       true,
	}

	res, err := cmd.Execute(context)
	if err != nil {
		return "", res.ExitCode, err
	}

	return res.Stdout, res.ExitCode, nil

}
