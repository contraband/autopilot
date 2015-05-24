package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/plugin"
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
	appRepo := NewApplicationRepo(cliConnection)
	appName, manifestPath, appPath, err := ParseArgs(args)
	fatalIf(err)

	venerableAppName := appName + "-venerable"

	err = appRepo.RenameApplication(appName, venerableAppName)
	fatalIf(err)

	err = appRepo.PushApplication(manifestPath, appPath)
	fatalIf(err)

	err = appRepo.DeleteApplication(venerableAppName)
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

func ParseArgs(args []string) (string, string, string, error) {
	flags := flag.NewFlagSet("zero-downtime-push", flag.ContinueOnError)
	manifestPath := flags.String("f", "", "path to an application manfiest")
	appPath := flags.String("p", "", "path to application files")

	err := flags.Parse(args[2:])
	if err != nil {
		return "", "", "", err
	}

	appName := args[1]

	if *manifestPath == "" {
		return "", "", "", ErrNoManifest
	}

	return appName, *manifestPath, *appPath, nil
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

func (repo *ApplicationRepo) PushApplication(manifestPath, appPath string) error {
	args := []string{"push", "-f", manifestPath}

	if appPath != "" {
		args = append(args, "-p", appPath)
	}

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
