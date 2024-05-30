package vals

import (
	"net/url"

	"github.com/apple/pkl-go/pkl"
	"github.com/helmfile/vals"
	"go.uber.org/zap"
)

func NewValsReader(logger *zap.Logger) (*ValsReader, error) {
	runtime, err := ValsInstance()

	if err != nil {
		return nil, err
	}

	return &ValsReader{
		Runtime: runtime,
		Logger:  logger,
	}, nil
}

type ValsReader struct {
	Runtime *vals.Runtime
	Logger  *zap.Logger
}

func (r *ValsReader) Read(url url.URL) ([]byte, error) {
	url.Scheme = ""
	code, err := r.Runtime.Get(url.String())

	if err != nil {
		return []byte{}, err
	}

	return []byte(code), nil
}

func (r *ValsReader) Scheme() string {
	return "vals"
}

func (r *ValsReader) IsGlobbable() bool {
	return false
}

func (r *ValsReader) HasHierarchicalUris() bool {
	return false
}

func (r *ValsReader) ListElements(url url.URL) ([]pkl.PathElement, error) {
	return make([]pkl.PathElement, 0), nil
}
