package mgr

import (
	"path/filepath"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mbark/punkt/conf"
	"github.com/mbark/punkt/file"
	"github.com/mbark/punkt/mgr/generic"
	"github.com/mbark/punkt/mgr/git"
	"github.com/mbark/punkt/mgr/symlink"
)

// ManagerConfig ...
type ManagerConfig struct {
	Symlinks []symlink.Symlink
}

// Manager ...
type Manager interface {
	Name() string
	Dump() (string, error)
	Ensure() error
	Update() error
}

// RootManager ...
type RootManager struct {
	LinkManager symlink.LinkManager
	config      conf.Config
}

// NewRootManager ...
func NewRootManager(config conf.Config) *RootManager {
	return &RootManager{
		LinkManager: symlink.NewLinkManager(config),
		config:      config,
	}
}

// All returns a list of all available managers
func (rootMgr RootManager) All() []Manager {
	var mgrs []Manager
	for name := range rootMgr.config.Managers {
		mgr := generic.NewManager(rootMgr.config, rootMgr.ConfigFile(name), name)
		mgrs = append(mgrs, mgr)
	}

	return append(mgrs, rootMgr.Git(), rootMgr.Symlink())
}

// Dump ...
func (rootMgr RootManager) Dump(mgrs []Manager) error {
	var result error
	for i := range mgrs {
		out, err := mgrs[i].Dump()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "dump failed [manager: %s]", mgrs[i].Name()))
			continue
		}

		err = file.Save(rootMgr.config.Fs, out, rootMgr.ConfigFile(mgrs[i].Name()))
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "failed to save configuration [manager: %s]", mgrs[i].Name()))
			continue
		}
	}

	return result
}

// Ensure ...
func (rootMgr RootManager) Ensure(mgrs []Manager) error {
	var result error
	for i := range mgrs {
		logger := logrus.WithField("manager", mgrs[i].Name())
		logger.Debug("running ensure")

		err := mgrs[i].Ensure()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "ensure failed [manager: %s]", mgrs[i].Name()))
			continue
		}

		symlinks, err := rootMgr.readSymlinks(mgrs[i].Name())
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "unable to get symlinks [manager: %s]", mgrs[i].Name()))
			continue
		}

		for i := range symlinks {
			expanded := rootMgr.LinkManager.Expand(symlinks[i])
			err = rootMgr.LinkManager.Ensure(expanded)
			if err != nil {
				result = multierror.Append(result, errors.Wrapf(err, "unable to ensure symlink [manager: %s, symlink: %v]", mgrs[i].Name(), symlinks[i]))
				continue
			}
		}
	}

	return result
}

// Update ...
func (rootMgr RootManager) Update(mgrs []Manager) error {
	var result error
	for i := range mgrs {
		err := mgrs[i].Update()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "update failed [manager: %s]", mgrs[i].Name()))
			continue
		}
	}

	return result
}

func (rootMgr RootManager) readSymlinks(name string) ([]symlink.Symlink, error) {
	var config ManagerConfig
	err := file.ReadToml(rootMgr.config.Fs, &config, rootMgr.ConfigFile(name))
	if err != nil && err != file.ErrNoSuchFile {
		if err == file.ErrNoSuchFile {
			return []symlink.Symlink{}, nil
		}

		return nil, err
	}

	return config.Symlinks, nil
}

// Git ...
func (rootMgr RootManager) Git() git.Manager {
	return *git.NewManager(rootMgr.config, rootMgr.ConfigFile("git"))
}

// Symlink ...
func (rootMgr RootManager) Symlink() symlink.Manager {
	return *symlink.NewManager(rootMgr.config, rootMgr.ConfigFile("symlink"))
}

// ConfigFile ...
func (rootMgr RootManager) ConfigFile(name string) string {
	return filepath.Join(rootMgr.config.PunktHome, name+".toml")
}
