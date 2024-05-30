package registry

const (
	// OCIScheme is the URL scheme for OCI-based requests
	OCIScheme = "oci"

	// CredentialsFileBasename is the filename for auth credentials file
	CredentialsFileBasename = "registry/config.json"

	// ConfigMediaType is the reserved media type for the Helm chart manifest config
	ConfigMediaType = "application/vnd.hpkl.io.config.v1+json"

	// ConfigMediaType is the reserved media type for the Helm chart manifest config
	MetadataMediaType = "application/vnd.hpkl.io.metadata.v1+json"

	// ChartLayerMediaType is the reserved media type for Helm chart package content
	PackageLayerMediaType = "application/vnd.hpkl.io.pkg.content.v1.tar+gzip"
)
