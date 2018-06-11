package symlink_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"gopkg.in/src-d/go-billy.v4/memfs"

	"github.com/mbark/punkt/conf"
	"github.com/mbark/punkt/file"
	"github.com/mbark/punkt/mgr/symlink"
	"github.com/mbark/punkt/mgr/symlink/symlinktest"
	"github.com/mbark/punkt/printer"
)

func TestSymlink(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Symlink Suite")
}

var _ = Describe("Symlink: Manager", func() {
	var config *conf.Config
	var linkMgr *symlinktest.MockLinkManager
	var mgr *symlink.Manager
	var configFile string
	var existingFile string

	BeforeEach(func() {
		logrus.SetLevel(logrus.PanicLevel)
		config = &conf.Config{
			UserHome:   "/home",
			PunktHome:  "/home/.config/punkt",
			Dotfiles:   "/home/.dotfiles",
			WorkingDir: "/home",
			Fs:         memfs.New(),
			Command:    fakeCommand,
		}
		printer.Log.Out = ioutil.Discard

		existingFile = filepath.Join(config.UserHome, ".configFile")
		_, err := config.Fs.Create(existingFile)
		Expect(err).To(BeNil())

		configFile = filepath.Join(config.PunktHome, "symlinks.toml")

		mgr = symlink.NewManager(*config, configFile)
		linkMgr = new(symlinktest.MockLinkManager)
		mgr.LinkManager = linkMgr

		linkMgr.On("New", mock.Anything, mock.Anything).Return(&symlink.Symlink{
			Target: "target",
			Link:   "link",
		})
		linkMgr.On("Ensure", mock.Anything).Return(nil)

	})

	It("should be called symlink", func() {
		Expect(mgr.Name()).To(Equal("symlink"))
	})

	var _ = Context("Dump", func() {
		It("should do nothing and always succeed", func() {
			out, err := mgr.Dump()
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	var _ = Context("Update", func() {
		It("should do nothing and always succeed", func() {
			Expect(mgr.Update()).To(Succeed())
		})
	})

	var _ = Context("Dump", func() {
		It("should do nothing and always succeed", func() {
			out, err := mgr.Dump()
			Expect(out).To(Equal(""))
			Expect(err).To(BeNil())
		})
	})

	var _ = Context("Add", func() {
		It("should make the target path absolute", func() {
			target := filepath.Base(existingFile)
			location := "/foo/bar"
			expected := filepath.Join(config.WorkingDir, target)

			_, err := mgr.Add(target, location)
			Expect(err).To(BeNil())

			linkMgr.AssertCalled(GinkgoT(), "New", location, expected)
		})

		It("should ensure the symlink exists", func() {
			linkMgr = new(symlinktest.MockLinkManager)
			mgr.LinkManager = linkMgr
			linkMgr.On("New", mock.Anything, mock.Anything).Return(&symlink.Symlink{
				Target: "target",
				Link:   "link",
			})
			linkMgr.On("Ensure", mock.Anything).Return(nil)

			_, err := mgr.Add(existingFile, "/some/where")
			Expect(err).To(BeNil())

			linkMgr.AssertCalled(GinkgoT(), "Ensure", mock.Anything)
		})

		It("should fail if the symlink can't be ensured", func() {
			linkMgr = new(symlinktest.MockLinkManager)
			mgr.LinkManager = linkMgr
			linkMgr.On("New", mock.Anything, mock.Anything).Return(&symlink.Symlink{
				Target: "target",
				Link:   "link",
			})
			linkMgr.On("Ensure", mock.Anything).Return(fmt.Errorf("fail"))

			_, err := mgr.Add(existingFile, "/location")
			Expect(err).NotTo(BeNil())
		})

		It("should save the symlink added", func() {
			s, err := mgr.Add(existingFile, "/some/where")
			Expect(err).To(BeNil())

			var c symlink.Config
			err = file.ReadToml(config.Fs, &c, configFile)
			Expect(err).To(BeNil())

			Expect(c.Symlinks).To(ConsistOf(*s))
		})

		It("should not save the symlink if it already exists", func() {
			_, err := mgr.Add(existingFile, "/some/where")
			Expect(err).To(BeNil())
			s, err := mgr.Add(existingFile, "/some/where")
			Expect(err).To(BeNil())

			var c symlink.Config
			err = file.ReadToml(config.Fs, &c, configFile)
			Expect(err).To(BeNil())

			Expect(c.Symlinks).To(ConsistOf(*s))
		})

		It("should fail if the stored config can't be parsed", func() {
			err := file.Save(config.Fs, "foo", configFile)
			Expect(err).To(BeNil())

			_, err = mgr.Add("/target", "/location")
			Expect(err).NotTo(BeNil())
		})

		It("should fail if the file to add doesn't exist", func() {
			_, err := mgr.Add("/a/file", "")
			Expect(err).NotTo(BeNil())
		})
	})

	var _ = Context("Remove", func() {
		It("should succeed when removing a link that was added", func() {
			s, err := mgr.Add(existingFile, "")
			Expect(err).To(BeNil())
			linkMgr.On("Remove", mock.Anything).Return(s, nil)

			err = mgr.Remove(existingFile)
			Expect(err).To(BeNil())

			var c symlink.Config
			err = file.ReadToml(config.Fs, &c, configFile)
			Expect(err).To(BeNil())

			Expect(c.Symlinks).To(BeEmpty())
		})

		It("should succeed if the config file doesn't exist", func() {
			linkMgr.On("Remove", mock.Anything).Return(new(symlink.Symlink), nil)
			Expect(mgr.Remove(existingFile)).To(Succeed())
		})

		It("should succeed even if the symlink isn't stored in the config file", func() {
			linkMgr.On("Remove", mock.Anything).Return(new(symlink.Symlink), nil)
			_, err := mgr.Add(existingFile, "/some/where")
			Expect(err).To(BeNil())

			Expect(mgr.Remove(existingFile)).To(Succeed())
		})

		It("should fail and not remove the link if it can't remove it", func() {
			linkMgr.On("Remove", mock.Anything).Return(new(symlink.Symlink), fmt.Errorf("fail"))
			s, err := mgr.Add(existingFile, "/some/where")
			Expect(err).To(BeNil())

			Expect(mgr.Remove(s.Link)).NotTo(Succeed())
		})

		It("should handle relative paths", func() {
			linkMgr.On("Remove", mock.Anything).Return(new(symlink.Symlink), nil)
			mgr.Add(existingFile, "")

			relPath, err := filepath.Rel(config.WorkingDir, existingFile)
			Expect(err).To(BeNil())

			Expect(mgr.Remove(relPath)).To(Succeed())

			linkMgr.AssertCalled(GinkgoT(), "Remove", existingFile)
		})

		It("should fail if the file doesn't exist", func() {
			Expect(mgr.Remove("/non/existant")).NotTo(Succeed())
		})
	})
})

func fakeCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestAddHelperProcess", "--", command}
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestAddHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	os.Exit(1)
}
