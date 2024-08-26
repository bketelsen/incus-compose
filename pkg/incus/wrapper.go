package incus

import (
	"context"
	"fmt"

	execute "github.com/alexellis/go-execute/v2"
)

var command = "incus"

// ExecuteShell executes the incus command in shell mode
func ExecuteShell(context context.Context, args []string) (string, error) {
	cmd := execute.ExecTask{
		Command:     command,
		Args:        args,
		StreamStdio: false,
		Shell:       true,
	}

	res, err := cmd.Execute(context)
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return res.Stdout, fmt.Errorf("error %d - %s", res.ExitCode, res.Stderr)
	}
	return res.Stdout, nil
}

// ExecuteShellStream executes the incus command in shell mode with stdio streaming
func ExecuteShellStream(context context.Context, args []string) (string, error) {
	cmd := execute.ExecTask{
		Command:     command,
		Args:        args,
		StreamStdio: true,
		Shell:       true,
	}

	res, err := cmd.Execute(context)
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return res.Stdout, fmt.Errorf("error %d - %s", res.ExitCode, res.Stderr)
	}

	return res.Stdout, nil
}

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
