package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAutopilot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Autopilot Suite")
}

var _ = Describe("Autopilot", func() {
	Describe("performing the zero-downtime deploy", func() {
		It("does the old switcheroo", func() {
			By("renaming the old application")
			By("pushing the new application")
			By("unmapping the routes from the old application")
			By("deleting the old application")
		})
	})
})
