package proxy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCoderExercise(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CoderExercise Suite")
}
