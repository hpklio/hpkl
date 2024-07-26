package vals

import (
	"net/url"

	"github.com/apple/pkl-go/pkl"
	"github.com/helmfile/vals"
)

func NewValsReader() (*ValsReader, error) {
	runtime, err := ValsInstance()

	if err != nil {
		return nil, err
	}

	return &ValsReader{
		Runtime: runtime,
	}, nil
}

type ValsReader struct {
	Runtime *vals.Runtime
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
