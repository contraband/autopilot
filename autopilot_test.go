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
		appName, manifestPath, appPath, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"appname",
				"-f", "./fixtures/manifests/manifest.yml",
				"-p", "app-path",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(appName).To(Equal("appname"))
		Expect(manifestPath).To(Equal("./fixtures/manifests/manifest.yml"))
		Expect(appPath).To(Equal("app-path"))
	})

	It("does not require path or manifest flag", func() {
		appName, _, _, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"appname",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(appName).To(Equal("appname"))
	})

	It("does not require app name if provided in the manifest", func() {
		appName, manifestPath, _, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"-f", "./fixtures/manifests/manifest.yml",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(appName).To(Equal("appname-from-manifest"))
		Expect(manifestPath).To(Equal("./fixtures/manifests/manifest.yml"))
	})

	It("errors if no app name is provided (as either a flag or in the manifest)", func() {
		_, _, _, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"-f", "./fixtures/manifests/manifest-without-appname.yml",
			},
		)
		Expect(err).To(MatchError(ErrNoAppName))
	})

	It("the provided app flag takes precedence over an app name in the manifest", func() {
		appName, manifestPath, _, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"appname-from-flag",
				"-f", "./fixtures/manifests/manifest.yml",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(appName).To(Equal("appname-from-flag"))
		Expect(manifestPath).To(Equal("./fixtures/manifests/manifest.yml"))
	})

	It("defaults to finding the name from the first app in the manifest", func() {
		appName, _, _, _ := ParseArgs(
			[]string{
				"zero-downtime-push",
				"-f", "./fixtures/manifests/manifest-with-multiple-apps.yml",
			},
		)
		Expect(appName).To(Equal("first-appname"))
	})

	It("errors if manifest path is bad", func() {
		_, _, _, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"-f", "./fixtures/manifests/nonexistent-manifest.yml",
			},
		)
		Expect(err).To(HaveOccurred())
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

	Describe("DoesAppExist", func() {

		It("returns an error if the cli returns an error", func() {
			cliConn.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("you shall not curl"))
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(MatchError("you shall not curl"))
		})

		It("returns an error if the cli response is invalid JSON", func() {
			response := []string{
				"}notjson{",
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(HaveOccurred())
		})

		It("returns an error if the cli response doesn't contain total_results", func() {
			response := []string{
				`{"brutal_results":2}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(MatchError("Missing total_results from api response"))
		})

		It("returns an error if the cli response contains a non-number total_results", func() {
			response := []string{
				`{"total_results":"sandwich"}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(MatchError("total_results didn't have a number sandwich"))
		})

		It("returns true if the app exists", func() {
			response := []string{
				`{"total_results":1}`,
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

			result, err := repo.DoesAppExist("app-name")

			Expect(cliConn.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
			args := cliConn.CliCommandWithoutTerminalOutputArgsForCall(0)
			Expect(args).To(Equal([]string{"curl", "v2/apps?q=name:app-name&q=space_guid:4"}))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("URL encodes the application name", func() {
			response := []string{
				`{"total_results":1}`,
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

			result, err := repo.DoesAppExist("app name")

			Expect(cliConn.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
			args := cliConn.CliCommandWithoutTerminalOutputArgsForCall(0)
			Expect(args).To(Equal([]string{"curl", "v2/apps?q=name:app+name&q=space_guid:4"}))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("returns false if the app does not exist", func() {
			response := []string{
				`{"total_results":0}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			result, err := repo.DoesAppExist("app-name")

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})

	})

	Describe("PushApplication", func() {
		It("pushes an application with both a manifest and a path", func() {
			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "/path/to/the/app")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"push",
				"appName",
				"-f", "/path/to/a/manifest.yml",
				"-p", "/path/to/the/app",
			}))
		})

		It("pushes an application with only a manifest", func() {
			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"push",
				"appName",
				"-f", "/path/to/a/manifest.yml",
			}))
		})

		It("returns errors from the push", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad app"))

			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "/path/to/the/app")
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
