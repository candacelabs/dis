package forward

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestForward(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Forward Suite")
}
