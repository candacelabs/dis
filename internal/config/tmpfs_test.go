package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResolveConfig and dis.tmpfs", func() {
	resolve := func(dockerfile string) *DisConfig {
		project := filepath.Join(GinkgoT().TempDir(), "memory-project")
		Expect(os.MkdirAll(filepath.Join(project, ".dis"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(project, ".dis", "Dockerfile"), []byte(dockerfile), 0o644)).To(Succeed())
		config, err := ResolveConfig(project)
		Expect(err).NotTo(HaveOccurred())
		return config
	}

	It("reads the label as a list of container paths, ignoring blanks", func() {
		config := resolve("FROM ubuntu:22.04\nLABEL dis.tmpfs=\"/tmp, /cache,\"\n")
		Expect(config.Tmpfs).To(Equal([]string{"/tmp", "/cache"}))
	})

	It("mounts nothing in memory without the label", func() {
		Expect(resolve("FROM ubuntu:22.04\n").Tmpfs).To(BeEmpty())
	})
})
