package rewind_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRewind(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rewind Suite")
}
