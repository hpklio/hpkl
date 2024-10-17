package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"hpkl.io/hpkl/pkg/logger"
)

func TestDeduplicate(t *testing.T) {
	stdWriter := new(bytes.Buffer)
	errWriter := new(bytes.Buffer)

	r, err := NewResolver(&AppConfig{
		Logger:    logger.New(stdWriter, errWriter),
		project:   nil,
		ctx:       context.Background(),
		PlainHttp: true,
	})

	if err != nil {
		t.Fatal(err)
	}

	original := map[string]*Metadata{
		"package://host/path@1.2.3":  {Name: "path", Version: "1.2.3", PackageUri: "package://host/path@1.2.3"},
		"package://host/path@1.2.4":  {Name: "path", Version: "1.2.4", PackageUri: "package://host/path@1.2.4"},
		"package://host/other@1.2.3": {Name: "other", Version: "1.2.3", PackageUri: "package://host/other@1.2.3"},
	}

	actual, err := r.Deduplicate(original)

	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]*Metadata{
		"package://host/path@1.2.4":  {Name: "path", Version: "1.2.4", PackageUri: "package://host/path@1.2.4"},
		"package://host/other@1.2.3": {Name: "other", Version: "1.2.3", PackageUri: "package://host/other@1.2.3"},
	}

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf(diff)
	}
}
