package pklutils

import (
	"github.com/apple/pkl-go/pkl"
	"hpkl.io/hpkl/pkg/logger"
	"hpkl.io/hpkl/pkg/vals"
)

func WithVals(logger *logger.Logger) func(options *pkl.EvaluatorOptions) {
	valsReader, err := vals.NewValsReader(logger)

	if err != nil {
		panic(err)
	}

	return func(options *pkl.EvaluatorOptions) {
		options.AllowedResources = append(options.AllowedResources, "vals:")
		options.ResourceReaders = append(options.ResourceReaders, valsReader)
	}
}
