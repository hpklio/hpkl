package vals

import (
	"bufio"
	"bytes"
	"sync"
)

const (
	// cache size for improving performance of ref+.* secrets rendering
	valsCacheSize = 512
)

var instance *KeysRuntime
var once sync.Once

func ValsInstance() (*KeysRuntime, error) {
	var err error

	var valsOutputBuffer bytes.Buffer
	valsOutput := bufio.NewWriter(&valsOutputBuffer)

	once.Do(func() {
		instance, err = New(
			Options{CacheSize: valsCacheSize, LogOutput: valsOutput, FailOnMissingKeyInMap: true},
		)
	})

	return instance, err
}
