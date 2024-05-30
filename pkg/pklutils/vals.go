package pklutils

import (
	"github.com/apple/pkl-go/pkl"
	"go.uber.org/zap"
	"hpkl.io/hpkl/pkg/vals"
)

func WithVals(logger *zap.Logger) func(options *pkl.EvaluatorOptions) {
	valsReader, err := vals.NewValsReader(logger)

	if err != nil {
		panic(err)
	}

	return func(options *pkl.EvaluatorOptions) {
		options.AllowedResources = append(options.AllowedResources, "vals:")
		options.ResourceReaders = append(options.ResourceReaders, valsReader)
		options.OutputFormat = "yaml"
	}
}
