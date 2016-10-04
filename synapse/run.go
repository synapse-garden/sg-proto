package main

import (
	"strings"

	"github.com/synapse-garden/sg-proto/client"
	"github.com/synapse-garden/sg-proto/cmd"
)

// RunWindow runs the SG client terminal.
func RunWindow(cli *client.Client) error {
	if err := cmd.Info(cli); err != nil {
		return err
	}

	for s := cli.State; s.Scan(); {
		com := cmd.GetCommand(strings.Split(s.Text(), " ")...)
		if err := com(cli); err == cmd.ErrQuit {
			return cmd.OutputString("Bye!")(cli)
		} else if err != nil {
			return err
		}
	}
	return nil
}
