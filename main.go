package main

import (
	"fmt"
	syslog "log"
	"os"

	"github.com/christophercampbell/bridge-connector/app"
	"github.com/urfave/cli/v2"
)

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = app.AppName
	cliApp.Commands = []*cli.Command{
		{
			Name:    "run",
			Aliases: []string{},
			Usage:   fmt.Sprintf("Run the %v", app.AppName),
			Action:  app.Run,
			Flags:   app.RunFlags,
		},
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		syslog.Fatal(err)
		os.Exit(1)
	}
}
