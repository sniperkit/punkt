package path

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-billy.v4"
)

// CreateNecessaryDirectories constructs the directories necessary to be able to
// write to the file
func CreateNecessaryDirectories(fs billy.Filesystem, file string) error {
	dir := filepath.Dir(file)
	logrus.WithField("dir", dir).Debug("Creating required directories")

	return fs.MkdirAll(dir, os.ModePerm)
}

// GetUserHome returns the user's home directory
func GetUserHome() string {
	usr, err := user.Current()
	if err != nil {
		logrus.WithError(err).Fatal("Unable to get user home")
		return ""
	}

	return usr.HomeDir
}

// ExpandHome takes the given string and replaces occurrences of ~ with the
// current user's home directory
func ExpandHome(s string, home string) string {
	return strings.Replace(s, "~", home, 1)
}

// UnexpandHome takes the given string and replaces the user's home with ~
// This is useful when you want to make something home-relative, rather than
// absolute
func UnexpandHome(s string, home string) string {
	return strings.Replace(s, home, "~", 1)
}
