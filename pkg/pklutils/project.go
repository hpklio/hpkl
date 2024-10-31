package pklutils

import (
	"context"
	"net/url"
	"os"
	"path/filepath"

	"github.com/apple/pkl-go/pkl"
)

func FileSource(pathElems ...string) *pkl.ModuleSource {
	src := filepath.Join(pathElems...)
	if !filepath.IsAbs(src) {
		p, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		src = filepath.Join(p, src)
	}

	// Fix windows path from C:\Project\Path to /C:/Project/Path
	if os.PathSeparator == '\\' {
		src = "/" + filepath.ToSlash(src)
	}

	return &pkl.ModuleSource{
		Uri: &url.URL{
			Scheme: "file",
			Path:   src,
		},
	}
}

func LoadProject(ctx context.Context, path string) (*pkl.Project, error) {
	ev, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return nil, err
	}
	return LoadProjectFromEvaluator(ctx, ev, path)
}

func LoadProjectFromEvaluator(context context.Context, ev pkl.Evaluator, path string) (*pkl.Project, error) {
	var proj pkl.Project

	if err := ev.EvaluateOutputValue(context, FileSource(path), &proj); err != nil {
		return nil, err
	}
	return &proj, nil
}
