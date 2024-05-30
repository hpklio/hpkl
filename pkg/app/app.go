package app

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"go.uber.org/zap"
)

type AppConfig struct {
	Logger  *zap.Logger
	Project *pkl.Project
}

func NewAppConfig(ctx context.Context) (*AppConfig, error) {
	logger, err := NewLogger()

	if err != nil {
		return nil, err
	}

	project, err := pkl.LoadProject(ctx, "PklProject")

	if err != nil {
		return nil, err
	}

	return &AppConfig{
		Logger:  logger,
		Project: project,
	}, nil
}
