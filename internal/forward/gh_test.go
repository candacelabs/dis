package forward

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/mount"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GHMounts", func() {
	var (
		home        string
		environment map[string]string
	)

	BeforeEach(func() {
		home = GinkgoT().TempDir()
		environment = map[string]string{}
		originalHome, originalGetenv := ghHomeDir, ghGetenv
		ghHomeDir = func() (string, error) { return home, nil }
		ghGetenv = func(key string) string { return environment[key] }
		DeferCleanup(func() { ghHomeDir, ghGetenv = originalHome, originalGetenv })
	})

	logIn := func(dir string) {
		Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "hosts.yml"), []byte("github.com:\n    oauth_token: gho_test\n"), 0o600)).To(Succeed())
	}

	It("mounts ~/.config/gh read-only and points GH_CONFIG_DIR at it", func() {
		logIn(filepath.Join(home, ".config", "gh"))
		mounts, env, err := GHMounts()
		Expect(err).NotTo(HaveOccurred())
		Expect(mounts).To(Equal([]mount.Mount{{Type: mount.TypeBind, Source: filepath.Join(home, ".config", "gh"), Target: "/dis/gh", ReadOnly: true}}))
		Expect(env).To(Equal([]string{"GH_CONFIG_DIR=/dis/gh"}))
	})

	It("uses the host's GH_CONFIG_DIR, as gh does, before XDG_CONFIG_HOME", func() {
		custom := filepath.Join(home, "elsewhere")
		logIn(custom)
		environment["GH_CONFIG_DIR"] = custom
		environment["XDG_CONFIG_HOME"] = filepath.Join(home, "xdg")
		mounts, _, err := GHMounts()
		Expect(err).NotTo(HaveOccurred())
		Expect(mounts).To(HaveExactElements(HaveField("Source", custom)))
	})

	It("uses $XDG_CONFIG_HOME/gh when GH_CONFIG_DIR is unset", func() {
		logIn(filepath.Join(home, "xdg", "gh"))
		environment["XDG_CONFIG_HOME"] = filepath.Join(home, "xdg")
		mounts, _, err := GHMounts()
		Expect(err).NotTo(HaveOccurred())
		Expect(mounts).To(HaveExactElements(HaveField("Source", filepath.Join(home, "xdg", "gh"))))
	})

	It("forwards nothing when the host has no gh login", func() {
		mounts, env, err := GHMounts()
		Expect(err).NotTo(HaveOccurred())
		Expect(mounts).To(BeEmpty())
		Expect(env).To(BeEmpty())
	})
})
