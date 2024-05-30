package registry

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	orascontext "oras.land/oras-go/pkg/context"
	"oras.land/oras-go/pkg/registry"
)

var immutableOciAnnotations = []string{
	ocispec.AnnotationVersion,
	ocispec.AnnotationTitle,
}

// ctx retrieves a fresh context.
// disable verbose logging coming from ORAS (unless debug is enabled)
func ctx(out io.Writer, debug bool) context.Context {
	if !debug {
		return orascontext.Background()
	}
	ctx := orascontext.WithLoggerFromWriter(context.Background(), out)
	orascontext.GetLogger(ctx).Logger.SetLevel(logrus.DebugLevel)
	return ctx
}

// parseReference will parse and validate the reference, and clean tags when
// applicable tags are only cleaned when plus (+) signs are present, and are
// converted to underscores (_) before pushing
// See https://github.com/helm/helm/issues/10166
func parseReference(raw string) (registry.Reference, error) {
	// The sole possible reference modification is replacing plus (+) signs
	// present in tags with underscores (_). To do this properly, we first
	// need to identify a tag, and then pass it on to the reference parser
	// NOTE: Passing immediately to the reference parser will fail since (+)
	// signs are an invalid tag character, and simply replacing all plus (+)
	// occurrences could invalidate other portions of the URI
	parts := strings.Split(raw, ":")
	if len(parts) > 1 && !strings.Contains(parts[len(parts)-1], "/") {
		tag := parts[len(parts)-1]

		if tag != "" {
			// Replace any plus (+) signs with known underscore (_) conversion
			newTag := strings.ReplaceAll(tag, "+", "_")
			raw = strings.ReplaceAll(raw, tag, newTag)
		}
	}

	return registry.ParseReference(raw)
}

// generateOCIAnnotations will generate OCI annotations to include within the OCI manifest
func generateOCIAnnotations(project *pkl.Project, creationTime string) map[string]string {

	// Get annotations from package attributes
	ociAnnotations := generatePackageOCIAnnotations(project, creationTime)

	return ociAnnotations
}

// getPackageOCIAnnotations will generate OCI annotations from the provided package
func generatePackageOCIAnnotations(project *pkl.Project, creationTime string) map[string]string {
	pkgOCIAnnotations := map[string]string{}

	pkgOCIAnnotations = addToMap(pkgOCIAnnotations, ocispec.AnnotationDescription, project.Package.Description)
	pkgOCIAnnotations = addToMap(pkgOCIAnnotations, ocispec.AnnotationTitle, project.Package.Name)
	pkgOCIAnnotations = addToMap(pkgOCIAnnotations, ocispec.AnnotationVersion, project.Package.Version)
	pkgOCIAnnotations = addToMap(pkgOCIAnnotations, ocispec.AnnotationURL, project.Package.BaseUri)

	if len(creationTime) == 0 {
		creationTime = time.Now().UTC().Format(time.RFC3339)
	}

	pkgOCIAnnotations = addToMap(pkgOCIAnnotations, ocispec.AnnotationCreated, creationTime)

	if len(project.Package.SourceCode) > 0 {
		pkgOCIAnnotations = addToMap(pkgOCIAnnotations, ocispec.AnnotationSource, project.Package.SourceCode)
	}

	if project.Package.Authors != nil && len(project.Package.Authors) > 0 {
		var maintainerSb strings.Builder

		for _, maintainer := range project.Package.Authors {

			if len(maintainer) > 0 {
				maintainerSb.WriteString(maintainer)
			}
		}

		pkgOCIAnnotations = addToMap(pkgOCIAnnotations, ocispec.AnnotationAuthors, maintainerSb.String())

	}

	return pkgOCIAnnotations
}

// addToMap takes an existing map and adds an item if the value is not empty
func addToMap(inputMap map[string]string, newKey string, newValue string) map[string]string {

	// Add item to map if its
	if len(strings.TrimSpace(newValue)) > 0 {
		inputMap[newKey] = newValue
	}

	return inputMap

}
