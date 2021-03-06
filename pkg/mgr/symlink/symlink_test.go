package symlink_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-billy.v4/memfs"

	"github.com/mbark/punkt/pkg/conf"
	"github.com/mbark/punkt/pkg/fs"
	"github.com/mbark/punkt/pkg/mgr/symlink"
	"github.com/mbark/punkt/pkg/printer"
)

func TestLinkManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Link Manager Suite")
}

var _ = Describe("Symlink: Link Manager", func() {
	var snapshot fs.Snapshot
	var config *conf.Config
	var mgr symlink.LinkManager

	BeforeEach(func() {
		logrus.SetLevel(logrus.PanicLevel)
		printer.Log.Out = ioutil.Discard
		snapshot = fs.Snapshot{Fs: memfs.New(), UserHome: "/home", WorkingDir: "/home"}
		config = &conf.Config{
			PunktHome: "/home/.config/punkt",
			Dotfiles:  "/home/.dotfiles",
		}

		mgr = symlink.NewLinkManager(*config, snapshot)
	})

	var _ = Context("New", func() {
		It("should return the link as is if both target and link are given", func() {
			expected := symlink.Symlink{Target: "/target", Link: "/link"}
			actual := mgr.New(expected.Target, expected.Link)

			Expect(*actual).To(Equal(expected))
		})

		It("should derive the target from the given link if possible", func() {
			link := filepath.Join(snapshot.UserHome, "/link")
			s := mgr.New("", link)

			Expect(s.Target).To(Equal(filepath.Join(config.Dotfiles, "/link")))
		})

		It("should derive the link from the given target if possible", func() {
			target := filepath.Join(config.Dotfiles, "/link")
			s := mgr.New(target, "")

			Expect(s.Link).To(Equal(filepath.Join(snapshot.UserHome, "/link")))
		})

		It("should keep the empty string if the link can't be derived", func() {
			s := mgr.New("/link", "")

			Expect(s.Link).To(Equal(""))
		})

		It("should keep an empty string if the link can't be made relative", func() {
			s := mgr.New("", ".")

			Expect(s.Target).To(Equal(""))
		})
	})

	var _ = Context("Remove", func() {
		var link string

		BeforeEach(func() {
			link = filepath.Join(snapshot.UserHome, "file")
			_, err := snapshot.Fs.Create(link)
			Expect(err).To(BeNil())
		})

		It("should remove the symlink if it exists", func() {
			s := mgr.New("", link)
			Expect(mgr.Ensure(s)).To(Succeed())

			_, err := mgr.Remove(s.Link)
			Expect(err).To(BeNil())

			_, err = snapshot.Fs.Readlink(link)
			Expect(err).NotTo(BeNil())
			_, err = snapshot.Fs.Stat(link)
			Expect(err).To(BeNil())
		})

		It("should fail if given link isn't a symlink", func() {
			_, err := mgr.Remove(link)
			Expect(err).NotTo(BeNil())
		})
	})

	var _ = Context("Ensure", func() {
		It("should succeed if the symlink already exists", func() {
			target := filepath.Join(config.Dotfiles, "target")
			_, err := snapshot.Fs.Create(target)
			Expect(err).To(BeNil())

			link := filepath.Join(snapshot.UserHome, "target")
			Expect(snapshot.Fs.Symlink(target, link)).To(Succeed())

			Expect(mgr.Ensure(&symlink.Symlink{Target: target, Link: link})).To(Succeed())
		})

		It("should handle when the file exists at link but not target", func() {
			link := filepath.Join(snapshot.UserHome, "target")
			_, err := snapshot.Fs.Create(link)
			Expect(err).To(BeNil())

			target := filepath.Join(config.Dotfiles, "target")

			Expect(mgr.Ensure(&symlink.Symlink{Target: target, Link: link})).To(Succeed())

			actual, err := snapshot.Fs.Readlink(link)
			Expect(err).To(BeNil())
			Expect(actual).To(Equal(target))
		})

		It("should handle when the target exists but not the link", func() {
			target := filepath.Join(config.Dotfiles, "target")
			_, err := snapshot.Fs.Create(target)
			Expect(err).To(BeNil())

			link := filepath.Join(snapshot.UserHome, "target")
			Expect(mgr.Ensure(&symlink.Symlink{Target: target, Link: link})).To(Succeed())

			actual, err := snapshot.Fs.Readlink(link)
			Expect(err).To(BeNil())
			Expect(actual).To(Equal(target))
		})

		It("should fail if the symlink can't be created", func() {
			link := filepath.Join(snapshot.UserHome, "target")
			_, err := snapshot.Fs.Create(link)
			Expect(err).To(BeNil())

			target := filepath.Join(config.Dotfiles, "target")
			_, err = snapshot.Fs.Create(target)
			Expect(err).To(BeNil())

			Expect(mgr.Ensure(&symlink.Symlink{Target: target, Link: link})).NotTo(Succeed())
		})

		It("should succeed even if neither of the two files exist", func() {
			link := &symlink.Symlink{Target: "/target", Link: "/link"}
			Expect(mgr.Ensure(link)).To(Succeed())
		})
	})

	var _ = Describe("Unexpand", func() {
		It("should expand tilde to the home directory", func() {
			s := mgr.Expand(symlink.Symlink{Target: "~/target", Link: "~/link"})
			Expect(s.Target).To(Equal(filepath.Join(snapshot.UserHome, "target")))
			Expect(s.Link).To(Equal(filepath.Join(snapshot.UserHome, "link")))
		})
	})

	var _ = Describe("Expand", func() {
		It("should unexpand the home directory to tilde", func() {
			s := mgr.Unexpand(symlink.Symlink{
				Target: filepath.Join(snapshot.UserHome, "target"),
				Link:   filepath.Join(snapshot.UserHome, "link"),
			})
			Expect(s.Target).To(Equal("~/target"))
			Expect(s.Link).To(Equal("~/link"))
		})
	})
})
