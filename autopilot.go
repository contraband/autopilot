package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/api/logs"
	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/contraband/autopilot/rewind"
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

func venerableAppName(appName string) string {
	return fmt.Sprintf("%s-venerable", appName)
}

func getActionsForApp(appRepo *ApplicationRepo, appName, manifestPath, appPath string, showLogs bool, keepOldApp bool) []rewind.Action {
	venName := venerableAppName(appName)
	var err error
	var curApp, venApp *AppEntity
	var haveVenToCleanup bool

	return []rewind.Action{
		// get info about current app
		{
			Forward: func() error {
				curApp, err = appRepo.GetAppMetadata(appName)
				if err != ErrAppNotFound {
					return err
				}
				curApp = nil
				return nil
			},
		},
		// get info about ven app
		{
			Forward: func() error {
				venApp, err = appRepo.GetAppMetadata(venName)
				if err != ErrAppNotFound {
					return err
				}
				venApp = nil
				return nil
			},
		},
		// rename any existing app such so that next step can push to a clear space
		{
			Forward: func() error {
				// Unless otherwise specified, go with our start state
				haveVenToCleanup = (venApp != nil)

				// If there is no current app running, that's great, we're done here
				if curApp == nil {
					return nil
				}

				// If current app isn't started, then we'll just delete it, and we're done
				if curApp.State != "STARTED" {
					return appRepo.DeleteApplication(appName)
				}

				// Do we have a ven app that will stop a rename?
				if venApp != nil {
					// Finally, since the current app claims to be healthy, we'll delete the venerable app, and rename the current over the top
					err = appRepo.DeleteApplication(venName)
					if err != nil {
						return err
					}
				}

				// Finally, rename
				haveVenToCleanup = true
				return appRepo.RenameApplication(appName, venName)
			},
		},
		// push
		{
			Forward: func() error {
				return appRepo.PushApplication(appName, manifestPath, appPath, showLogs)
			},
			ReversePrevious: func() error {
				if !haveVenToCleanup {
					return nil
				}

				// If the app cannot start we'll have a lingering application
				// We delete this application so that the rename can succeed
				appRepo.DeleteApplication(appName)

				return appRepo.RenameApplication(venName, appName)
			},
		},
		// delete
		{
			Forward: func() error {
				if !haveVenToCleanup || keepOldApp {
					return nil
				}
				return appRepo.DeleteApplication(venName)
			},
		},
	}
}

func getActionsForNewApp(appRepo *ApplicationRepo, appName, manifestPath, appPath string, showLogs bool) []rewind.Action {
	return []rewind.Action{
		// push
		{
			Forward: func() error {
				return appRepo.PushApplication(appName, manifestPath, appPath, showLogs)
			},
		},
	}
}

func (plugin AutopilotPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	// only handle if actually invoked, else it can't be uninstalled cleanly
	if args[0] != "zero-downtime-push" {
		return
	}

	appRepo := NewApplicationRepo(cliConnection)
	appName, manifestPath, appPath, showLogs, keepOldApp, err := ParseArgs(args)
	fatalIf(err)

	fatalIf((&rewind.Actions{
		Actions:              getActionsForApp(appRepo, appName, manifestPath, appPath, showLogs, keepOldApp),
		RewindFailureMessage: "Oh no. Something's gone wrong. I've tried to roll back but you should check to see if everything is OK.",
	}).Execute())

	fmt.Println()
	fmt.Println("A new version of your application has successfully been pushed!")
	fmt.Println()

	_ = appRepo.ListApplications()
}

func (AutopilotPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "autopilot",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 0,
			Build: 6,
		},
		Commands: []plugin.Command{
			{
				Name:     "zero-downtime-push",
				HelpText: "Perform a zero-downtime push of an application over the top of an old one",
				UsageDetails: plugin.Usage{
					Usage: "$ cf zero-downtime-push application-to-replace \\ \n \t-f path/to/new_manifest.yml \\ \n \t-p path/to/new/path",
				},
			},
		},
	}
}

func ParseArgs(args []string) (string, string, string, bool, bool, error) {
	flags := flag.NewFlagSet("zero-downtime-push", flag.ContinueOnError)
	manifestPath := flags.String("f", "", "path to an application manifest")
	appPath := flags.String("p", "", "path to application files")
	showLogs := flags.Bool("show-app-log", false, "tail and show application log during application start")
	keepOldApp := flags.Bool("keep-old-app", false, "do not remove the old application")

	if len(args) < 2 || strings.HasPrefix(args[1], "-") {
		return "", "", "", false, true, ErrNoArgs
	}
	err := flags.Parse(args[2:])
	if err != nil {
		return "", "", "", false, true, err
	}

	appName := args[1]

	if *manifestPath == "" {
		return "", "", "", false, true, ErrNoManifest
	}

	return appName, *manifestPath, *appPath, *showLogs, *keepOldApp, nil
}

var (
	ErrNoArgs     = errors.New("app name must be specified")
	ErrNoManifest = errors.New("a manifest is required to push this application")
)

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

func (repo *ApplicationRepo) PushApplication(appName, manifestPath, appPath string, showLogs bool) error {
	args := []string{"push", appName, "-f", manifestPath, "--no-start"}

	if appPath != "" {
		args = append(args, "-p", appPath)
	}

	_, err := repo.conn.CliCommand(args...)
	if err != nil {
		return err
	}

	if showLogs {
		app, err := repo.conn.GetApp(appName)
		if err != nil {
			return err
		}
		dopplerEndpoint, err := repo.conn.DopplerEndpoint()
		if err != nil {
			return err
		}
		token, err := repo.conn.AccessToken()
		if err != nil {
			return err
		}

		cons := consumer.New(dopplerEndpoint, nil, nil)
		defer cons.Close()

		messages, errors := cons.TailingLogs(app.Guid, token)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			for {
				select {
				case m := <-messages:
					if m.GetSourceType() != "STG" { // skip STG messages as the cf tool already prints them
						os.Stderr.WriteString(logs.NewNoaaLogMessage(m).ToLog(time.Local) + "\n")
					}
				case e := <-errors:
					log.Println("error reading logs:", e)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	_, err = repo.conn.CliCommand("start", appName)
	if err != nil {
		return err
	}

	return nil
}

func (repo *ApplicationRepo) DeleteApplication(appName string) error {
	_, err := repo.conn.CliCommand("delete", appName, "-f")
	return err
}

func (repo *ApplicationRepo) ListApplications() error {
	_, err := repo.conn.CliCommand("apps")
	return err
}

type AppEntity struct {
	State string `json:"state"`
}

var (
	ErrAppNotFound = errors.New("application not found")
)

// GetAppMetadata returns metadata about an app with appName
func (repo *ApplicationRepo) GetAppMetadata(appName string) (*AppEntity, error) {
	space, err := repo.conn.GetCurrentSpace()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf(`v2/apps?q=name:%s&q=space_guid:%s`, url.QueryEscape(appName), space.Guid)
	result, err := repo.conn.CliCommandWithoutTerminalOutput("curl", path)

	if err != nil {
		return nil, err
	}

	jsonResp := strings.Join(result, "")

	output := struct {
		Resources []struct {
			Entity AppEntity `json:"entity"`
		} `json:"resources"`
	}{}
	err = json.Unmarshal([]byte(jsonResp), &output)

	if err != nil {
		return nil, err
	}

	if len(output.Resources) == 0 {
		return nil, ErrAppNotFound
	}

	return &output.Resources[0].Entity, nil
}
