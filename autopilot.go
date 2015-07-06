package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/concourse/autopilot/rewind"
)

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
		os.Exit(1)
	}
}

func main() {
	plugin.Start(&AutopilotPlugin{})
}

type AutopilotPlugin struct{}

func (plugin AutopilotPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	var err error
	appRepo := NewApplicationRepo(cliConnection)
	appName, argList := ParseArgs(args)
	venerableAppName := appName + "-venerable"

	actions := rewind.Actions{
		Actions: []rewind.Action{
			// rename
			{
				Forward: func() error {
					return appRepo.RenameApplication(appName, venerableAppName)
				},
			},

			// push
			{
				Forward: func() error {
					return appRepo.PushApplication(argList)
				},
				ReversePrevious: func() error {
					return appRepo.RenameApplication(venerableAppName, appName)
				},
			},

			// delete
			{
				Forward: func() error {
					return appRepo.DeleteApplication(venerableAppName)
				},
			},
		},
		RewindFailureMessage: "Oh no. Something's gone wrong. I've tried to roll back but you should check to see if everything is OK.",
	}

	err = actions.Execute()
	fatalIf(err)

	fmt.Println()
	fmt.Println("A new version of your application has successfully been pushed!")
	fmt.Println()

	err = appRepo.ListApplications()
	fatalIf(err)
}

func (AutopilotPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "autopilot",
		Commands: []plugin.Command{
			{
				Name:     "zero-downtime-push",
				HelpText: "Perform a zero-downtime push of an application over the top of an old one",
			},
		},
	}
}

func ParseArgs(args []string) (string, []string) {
	args[0] = "push"
	appName := args[1]
	return appName, args
}

var ErrNoManifest = errors.New("a manifest is required to push this application")

type ApplicationRepo struct {
	conn plugin.CliConnection
}

func NewApplicationRepo(conn plugin.CliConnection) *ApplicationRepo {
	return &ApplicationRepo{
		conn: conn,
	}
}

func (repo *ApplicationRepo) RenameApplication(oldName, newName string) error {
	_, err := repo.conn.CliCommand("rename", oldName, newName)
	return err
}

func (repo *ApplicationRepo) PushApplication(args []string) error {
	_, err := repo.conn.CliCommand(args...)
	return err
}

func (repo *ApplicationRepo) DeleteApplication(appName string) error {
	_, err := repo.conn.CliCommand("delete", appName, "-f")
	return err
}

func (repo *ApplicationRepo) ListApplications() error {
	_, err := repo.conn.CliCommand("apps")
	return err
}
