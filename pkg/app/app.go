package app

import (
	"context"
	"path"

	"github.com/apple/pkl-go/pkl"
	"go.uber.org/zap"
)

type AppConfig struct {
	Logger     *zap.Logger
	project    *pkl.Project
	ctx        context.Context
	PlainHttp  bool
	CacheDir   string
	WorkingDir string
	RootDir    string
}

func (a *AppConfig) Project() *pkl.Project {
	projectFile := path.Join(a.WorkingDir, "PklProject")

	if a.project == nil {
		proj, err := pkl.LoadProject(a.ctx, projectFile)
		if err != nil {
			panic(err)
		}
		a.project = proj
	}
	return a.project
}

func NewAppConfig(ctx context.Context) (*AppConfig, error) {
	logger, err := NewLogger()

	if err != nil {
		return nil, err
	}

	return &AppConfig{
		Logger: logger,
		ctx:    ctx,
	}, nil
}
