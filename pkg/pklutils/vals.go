package pklutils

import (
	"github.com/apple/pkl-go/pkl"
	"hpkl.io/hpkl/pkg/vals"
)

func WithVals() func(options *pkl.EvaluatorOptions) {
	valsReader, err := vals.NewValsReader()

	if err != nil {
		panic(err)
	}

	return func(options *pkl.EvaluatorOptions) {
		options.AllowedResources = append(options.AllowedResources, "vals:")
		options.ResourceReaders = append(options.ResourceReaders, valsReader)
	}
}
