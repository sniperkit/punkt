package symlink

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mbark/punkt/conf"
	"github.com/mbark/punkt/file"
	"github.com/mbark/punkt/path"
	"github.com/mbark/punkt/printer"
)

// Manager ...
type Manager struct {
	LinkManager LinkManager
	configFile  string
	config      conf.Config
}

// Symlink describes a symlink, i.e. what it links from and what it links to
type Symlink struct {
	Target string
	Link   string
}

// Config ...
type Config struct {
	Symlinks []Symlink
}

func (symlink Symlink) String() string {
	return fmt.Sprintf("%s -> %s", symlink.Link, symlink.Target)
}

// NewManager ...
func NewManager(c conf.Config, configFile string) *Manager {
	return &Manager{
		LinkManager: NewLinkManager(c),
		config:      c,
		configFile:  configFile,
	}
}

// Add ...
func (mgr Manager) Add(target, newLocation string) (*Symlink, error) {
	absTarget, err := path.AsAbsolute(mgr.config.Fs, mgr.config.WorkingDir, target)
	if err != nil {
		printer.Log.Error("target file or directory does not exist: {fg 1}%s", target)
		return nil, err
	}

	symlink := mgr.LinkManager.New(newLocation, absTarget)
	err = mgr.LinkManager.Ensure(symlink)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to ensure %s exists", symlink)
	}

	storedLink, err := mgr.addToConfiguration(symlink)
	if err == nil {
		printer.Log.Success("symlink added: {fg 2}%s", storedLink)
	}

	return symlink, err
}

// Remove ...
func (mgr Manager) Remove(link string) error {
	absLink, err := path.AsAbsolute(mgr.config.Fs, mgr.config.WorkingDir, link)
	if err != nil {
		printer.Log.Error("file does not exist: {fg 1}%s", link)
		return err
	}

	config, err := mgr.readConfiguration()
	if err != nil && err != file.ErrNoSuchFile {
		printer.Log.Error("unable to read configuration file, error was: {fg 1}%s", err)
	}

	link = path.UnexpandHome(absLink, mgr.config.UserHome)
	var matchingSymlink *Symlink
	for _, symlink := range config.Symlinks {
		if symlink.Link == link {
			printer.Log.Success("found in configuration file, target is: {fg 2}%s", symlink.Target)
			matchingSymlink = &symlink
		}
	}

	if matchingSymlink == nil {
		printer.Log.Error("unable to find symlink in configuration file: {fg 1}%s", path.UnexpandHome(mgr.configFile, mgr.config.UserHome))
		return errors.Errorf("unable to find link %s in configuration", link)
	}

	symlink := mgr.LinkManager.Expand(*matchingSymlink)

	s, err := mgr.LinkManager.Remove(absLink, symlink.Target)
	if err != nil {
		printer.Log.Error("failed to remove link, error was: {fg 1}%s", err)
		err = errors.Wrapf(err, "failed to remove link %s", link)
		return err
	}

	removedLink, err := mgr.removeFromConfiguration(*s)
	if err == nil {
		printer.Log.Success("symlink removed: {fg 2}%s", removedLink)
	}

	return err
}

func (mgr Manager) readConfiguration() (Config, error) {
	var savedConfig Config
	err := file.ReadToml(mgr.config.Fs, &savedConfig, mgr.configFile)
	if err != nil {
		logger := logrus.WithField("configFile", mgr.configFile).WithError(err)
		if err == file.ErrNoSuchFile {
			location := path.UnexpandHome(mgr.configFile, mgr.config.UserHome)
			printer.Log.Note("no symlink configuration file at {fg 5}%s", location)
			logger.Warn("no configuration file found")
		} else {
			logger.Error("unable to read symlink configuration file")
		}
	}

	return savedConfig, err
}

func (mgr Manager) addToConfiguration(new *Symlink) (*Symlink, error) {
	logrus.WithField("newSymlink", new).Info("Storing symlink in configuration")
	saved, err := mgr.readConfiguration()
	if err != nil && err != file.ErrNoSuchFile {
		return nil, err
	}

	unexpanded := mgr.LinkManager.Unexpand(*new)
	for _, existing := range saved.Symlinks {
		if unexpanded.Target == existing.Target && unexpanded.Link == existing.Link {
			printer.Log.Note("symlink is already stored")
			logrus.WithField("symlink", unexpanded).Info("symlink already saved, nothing new to store")
			return unexpanded, nil
		}
	}

	saved.Symlinks = append(saved.Symlinks, *unexpanded)

	logrus.WithField("symlinks", saved).Debug("storing updated list of symlinks")
	return unexpanded, file.SaveToml(mgr.config.Fs, saved, mgr.configFile)
}

func (mgr Manager) removeFromConfiguration(symlink Symlink) (*Symlink, error) {
	var config Config
	err := file.ReadToml(mgr.config.Fs, &config, mgr.configFile)
	if err == file.ErrNoSuchFile {
		logrus.WithFields(logrus.Fields{
			"configFile": mgr.configFile,
		}).WithError(err).Warn("no configuration file found, configuration won't be updated")
		// TODO: return a special error for this case
		return nil, nil
	}

	unexpanded := mgr.LinkManager.Unexpand(symlink)
	index := -1
	for i, s := range config.Symlinks {
		logrus.WithFields(logrus.Fields{
			"unexpanded": unexpanded,
			"saved":      s,
		}).Debug("comparing if symlinks are the same")
		if unexpanded.Target == s.Target && unexpanded.Link == s.Link {
			index = i
		}
	}

	if index < 0 {
		logrus.WithFields(logrus.Fields{
			"symlink": symlink,
			"config":  config,
		}).Warn("symlink not found in configuration, not removing")
		return unexpanded, nil
	}

	config.Symlinks = append(config.Symlinks[:index], config.Symlinks[index+1:]...)
	return unexpanded, file.SaveToml(mgr.config.Fs, config, mgr.configFile)
}

// Name ...
func (mgr Manager) Name() string {
	return "symlink"
}

// Dump ...
func (mgr Manager) Dump() (string, error) { return "", nil }

// Update ...
func (mgr Manager) Update() error { return nil }

// Ensure ...
func (mgr Manager) Ensure() error { return nil }
