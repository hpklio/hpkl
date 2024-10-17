package vals

import (
	"net/url"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"hpkl.io/hpkl/pkg/logger"
)

const key_eparator = "!"

type MapElement struct {
	name        string
	isDirectory bool
}

func (m *MapElement) Name() string {
	return m.name
}

func (m *MapElement) IsDirectory() bool {
	return m.isDirectory
}

func NewValsReader(logger *logger.Logger) (*ValsReader, error) {
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
	Runtime *KeysRuntime
	Logger  *logger.Logger
}

func (r *ValsReader) IsGlobbable() bool {
	return true
}

func (r *ValsReader) HasHierarchicalUris() bool {
	return false
}

func (r *ValsReader) ListElements(url url.URL) ([]pkl.PathElement, error) {
	url.Scheme = ""
	basePart := strings.TrimSuffix(url.String(), "/**")
	key := strings.Replace(basePart, key_eparator, "#", 1)

	res, err := r.Runtime.GetMap(key)

	if err != nil {
		return nil, err
	}

	result := make([]pkl.PathElement, len(res))

	i := 0
	for k, v := range res {
		path := basePart + "/" + k
		switch v.(type) {
		case string:
			result[i] = &MapElement{path, false}
		default:
			result[i] = &MapElement{path, true}
		}
		i++
	}

	return result, nil
}

func (r *ValsReader) Read(url url.URL) ([]byte, error) {

	url.Scheme = ""
	key := strings.Replace(url.String(), key_eparator, "#", 1)

	res, err := r.Runtime.GetString(key)

	if err != nil {
		return nil, err
	} else {
		return []byte(res), nil
	}
}

func (r *ValsReader) Scheme() string {
	return "vals"
}
