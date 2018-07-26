package main_test

import (
	"errors"
	"testing"

	"code.cloudfoundry.org/cli/plugin/pluginfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/contraband/autopilot"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"
)

func TestAutopilot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Autopilot Suite")
}

var _ = Describe("Flag Parsing", func() {
	It("parses a complete set of args", func() {
		appName, manifestPath, appPath, showLogs, keepOldApp, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"appname",
				"-f", "manifest-path",
				"-p", "app-path",
				"-keep-old-app",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(appName).To(Equal("appname"))
		Expect(manifestPath).To(Equal("manifest-path"))
		Expect(appPath).To(Equal("app-path"))
		Expect(showLogs).To(Equal(false))
		Expect(keepOldApp).To(Equal(true))
	})

	It("requires a manifest", func() {
		_, _, _, _, _, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"appname",
				"-p", "app-path",
			},
		)
		Expect(err).To(MatchError(ErrNoManifest))
	})
})

var _ = Describe("ApplicationRepo", func() {
	var (
		cliConn *pluginfakes.FakeCliConnection
		repo    *ApplicationRepo
	)

	BeforeEach(func() {
		cliConn = &pluginfakes.FakeCliConnection{}
		repo = NewApplicationRepo(cliConn)
	})

	Describe("RenameApplication", func() {
		It("renames the application", func() {
			err := repo.RenameApplication("old-name", "new-name")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{"rename", "old-name", "new-name"}))
		})

		It("returns an error if one occurs", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("no app"))

			err := repo.RenameApplication("old-name", "new-name")
			Expect(err).To(MatchError("no app"))
		})
	})

	Describe("GetAppMetadata", func() {

		It("returns an error if the cli returns an error", func() {
			cliConn.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("you shall not curl"))
			_, err := repo.GetAppMetadata("app-name")

			Expect(err).To(MatchError("you shall not curl"))
		})

		It("returns an error if the cli response is invalid JSON", func() {
			response := []string{
				"}notjson{",
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.GetAppMetadata("app-name")

			Expect(err).To(HaveOccurred())
		})

		It("returns app data if the app exists", func() {
			response := []string{
				`{"resources":[{"entity":{"state":"STARTED"}}]}`,
			}
			spaceGUID := "4"

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			cliConn.GetCurrentSpaceReturns(
				plugin_models.Space{
					SpaceFields: plugin_models.SpaceFields{
						Guid: spaceGUID,
					},
				},
				nil,
			)

			result, err := repo.GetAppMetadata("app-name")

			Expect(cliConn.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
			args := cliConn.CliCommandWithoutTerminalOutputArgsForCall(0)
			Expect(args).To(Equal([]string{"curl", "v2/apps?q=name:app-name&q=space_guid:4"}))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
		})

		It("URL encodes the application name", func() {
			response := []string{
				`{"resources":[{"entity":{"state":"STARTED"}}]}`,
			}
			spaceGUID := "4"

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			cliConn.GetCurrentSpaceReturns(
				plugin_models.Space{
					SpaceFields: plugin_models.SpaceFields{
						Guid: spaceGUID,
					},
				},
				nil,
			)

			result, err := repo.GetAppMetadata("app name")

			Expect(cliConn.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
			args := cliConn.CliCommandWithoutTerminalOutputArgsForCall(0)
			Expect(args).To(Equal([]string{"curl", "v2/apps?q=name:app+name&q=space_guid:4"}))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
		})

		It("returns nil if the app does not exist", func() {
			response := []string{
				`{"resources":[]}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			result, err := repo.GetAppMetadata("app-name")

			Expect(err).To(Equal(ErrAppNotFound))
			Expect(result).To(BeNil())
		})

	})

	Describe("PushApplication", func() {
		It("pushes an application with both a manifest and a path", func() {
			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "/path/to/the/app", false)
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(2))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"push",
				"appName",
				"-f", "/path/to/a/manifest.yml",
				"--no-start",
				"-p", "/path/to/the/app",
			}))
		})

		It("pushes an application with only a manifest", func() {
			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "", false)
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(2))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"push",
				"appName",
				"-f", "/path/to/a/manifest.yml",
				"--no-start",
			}))
		})

		It("returns errors from the push", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad app"))

			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "/path/to/the/app", false)
			Expect(err).To(MatchError("bad app"))
		})
	})

	Describe("DeleteApplication", func() {
		It("deletes all trace of an application", func() {
			err := repo.DeleteApplication("app-name")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"delete", "app-name",
				"-f",
			}))
		})

		It("returns errors from the delete", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad app"))

			err := repo.DeleteApplication("app-name")
			Expect(err).To(MatchError("bad app"))
		})
	})

	Describe("ListApplications", func() {
		It("lists all the applications", func() {
			err := repo.ListApplications()
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{"apps"}))
		})

		It("returns errors from the list", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad apps"))

			err := repo.ListApplications()
			Expect(err).To(MatchError("bad apps"))
		})
	})
})
