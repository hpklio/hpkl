package app

import (
	"context"
	"io"
	"os"
	"path"

	"github.com/apple/pkl-go/pkl"
)

type AppConfig struct {
	Logger          *Logger
	project         *pkl.Project
	ctx             context.Context
	PlainHttp       bool
	CacheDir        string
	DefaultCacheDir string
	WorkingDir      string
	RootDir         string
	Parameters      []string
}

const (
	configPath = ".hpkl/config.pkl"
)

func (a *AppConfig) Project() *pkl.Project {

	projectFile := path.Join(a.WorkingDir, "PklProject")

	if a.project == nil {
		if _, err := os.Stat(projectFile); err == nil {

			proj, err := pkl.LoadProject(a.ctx, projectFile)

			if err != nil {
				a.Logger.Fatal("PklProject file not found in the working directory %s", a.WorkingDir)
			}
			a.project = proj
		} else {
			a.Logger.Fatal("PklProject file not found in the working directory %s", a.WorkingDir)
		}
	}

	return a.project
}

func (a *AppConfig) Reset() {
	a.project = nil
}

func NewAppConfig(ctx context.Context, outWriter io.Writer, errWriter io.Writer) (*AppConfig, error) {

	logger := NewLogger(outWriter, errWriter)

	return &AppConfig{
		Logger: logger,
		ctx:    ctx,
	}, nil
}
