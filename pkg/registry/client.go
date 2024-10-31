package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/pkg/errors"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/Masterminds/semver/v3"
	"github.com/containerd/containerd/remotes"

	"oras.land/oras-go/pkg/auth"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
	"oras.land/oras-go/pkg/registry"

	dockerauth "oras.land/oras-go/pkg/auth/docker"
	registryremote "oras.land/oras-go/pkg/registry/remote"
	registryauth "oras.land/oras-go/pkg/registry/remote/auth"
)

const registryUnderscoreMessage = `
OCI artifact references (e.g. tags) do not support the plus sign (+). To support
storing semantic versions, Helm adopts the convention of changing plus (+) to
an underscore (_) in package version tags when pushing to a registry and back to
a plus (+) when pulling from a registry.`

type (
	// Client works with OCI-compliant registries
	Client struct {
		debug       bool
		enableCache bool
		// path to repository config file e.g. ~/.docker/config.json
		credentialsFile    string
		out                io.Writer
		authorizer         auth.Client
		registryAuthorizer *registryauth.Client
		resolver           func(ref registry.Reference) (remotes.Resolver, error)
		httpClient         *http.Client
		plainHTTP          bool
	}

	// ClientOption allows specifying various settings configurable by the user for overriding the defaults
	// used when creating a new default client
	ClientOption func(*Client)
)

func WithPlainHttp(plain bool) ClientOption {
	return func(client *Client) {
		client.plainHTTP = plain
	}
}

func NewClient(options ...ClientOption) (*Client, error) {
	client := &Client{
		out: io.Discard,
	}
	for _, option := range options {
		option(client)
	}
	if client.credentialsFile == "" {
		home, err := os.UserHomeDir()

		if err != nil {
			return nil, err
		}
		client.credentialsFile = filepath.Join(home, ".docker", "config.json")
	}
	if client.authorizer == nil {
		authClient, err := dockerauth.NewClientWithDockerFallback(client.credentialsFile)
		if err != nil {
			return nil, err
		}
		client.authorizer = authClient
	}

	resolverFn := client.resolver // copy for avoiding recursive call
	client.resolver = func(ref registry.Reference) (remotes.Resolver, error) {
		if resolverFn != nil {
			// validate if the resolverFn returns a valid resolver
			if resolver, err := resolverFn(ref); resolver != nil && err == nil {
				return resolver, nil
			}
		}
		headers := http.Header{}
		// headers.Set("User-Agent", version.GetUserAgent())
		opts := []auth.ResolverOption{auth.WithResolverHeaders(headers)}
		if client.httpClient != nil {
			opts = append(opts, auth.WithResolverClient(client.httpClient))
		}
		if client.plainHTTP {
			opts = append(opts, auth.WithResolverPlainHTTP())
		}
		resolver, err := client.authorizer.ResolverWithOpts(opts...)
		if err != nil {
			return nil, err
		}
		return resolver, nil
	}

	// allocate a cache if option is set
	var cache registryauth.Cache
	if client.enableCache {
		cache = registryauth.DefaultCache
	}
	if client.registryAuthorizer == nil {
		client.registryAuthorizer = &registryauth.Client{
			Client: client.httpClient,
			// Header: http.Header{
			// 	"User-Agent": {version.GetUserAgent()},
			// },
			Cache: cache,
			Credential: func(_ context.Context, reg string) (registryauth.Credential, error) {
				dockerClient, ok := client.authorizer.(*dockerauth.Client)
				if !ok {
					return registryauth.EmptyCredential, errors.New("unable to obtain docker client")
				}

				username, password, err := dockerClient.Credential(reg)
				if err != nil {
					return registryauth.EmptyCredential, errors.New("unable to retrieve credentials")
				}

				// A blank returned username and password value is a bearer token
				if username == "" && password != "" {
					return registryauth.Credential{
						RefreshToken: password,
					}, nil
				}

				return registryauth.Credential{
					Username: username,
					Password: password,
				}, nil

			},
		}

	}
	return client, nil
}

// ClientOptDebug returns a function that sets the debug setting on client options set
func ClientOptDebug(debug bool) ClientOption {
	return func(client *Client) {
		client.debug = debug
	}
}

// ClientOptEnableCache returns a function that sets the enableCache setting on a client options set
func ClientOptEnableCache(enableCache bool) ClientOption {
	return func(client *Client) {
		client.enableCache = enableCache
	}
}

// ClientOptWriter returns a function that sets the writer setting on client options set
func ClientOptWriter(out io.Writer) ClientOption {
	return func(client *Client) {
		client.out = out
	}
}

// ClientOptCredentialsFile returns a function that sets the credentialsFile setting on a client options set
func ClientOptCredentialsFile(credentialsFile string) ClientOption {
	return func(client *Client) {
		client.credentialsFile = credentialsFile
	}
}

// ClientOptHTTPClient returns a function that sets the httpClient setting on a client options set
func ClientOptHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

func ClientOptPlainHTTP() ClientOption {
	return func(c *Client) {
		c.plainHTTP = true
	}
}

// ClientOptResolver returns a function that sets the resolver setting on a client options set
func ClientOptResolver(resolver remotes.Resolver) ClientOption {
	return func(client *Client) {
		client.resolver = func(_ registry.Reference) (remotes.Resolver, error) {
			return resolver, nil
		}
	}
}

type (
	// LoginOption allows specifying various settings on login
	LoginOption func(*loginOperation)

	loginOperation struct {
		username string
		password string
		insecure bool
		certFile string
		keyFile  string
		caFile   string
	}
)

// Login logs into a registry
func (c *Client) Login(host string, options ...LoginOption) error {
	operation := &loginOperation{}
	for _, option := range options {
		option(operation)
	}
	authorizerLoginOpts := []auth.LoginOption{
		auth.WithLoginContext(ctx(c.out, c.debug)),
		auth.WithLoginHostname(host),
		auth.WithLoginUsername(operation.username),
		auth.WithLoginSecret(operation.password),
		// auth.WithLoginUserAgent(version.GetUserAgent()),
		auth.WithLoginTLS(operation.certFile, operation.keyFile, operation.caFile),
	}
	if operation.insecure {
		authorizerLoginOpts = append(authorizerLoginOpts, auth.WithLoginInsecure())
	}
	if err := c.authorizer.LoginWithOpts(authorizerLoginOpts...); err != nil {
		return err
	}

	fmt.Fprintln(c.out, "Login Succeeded")
	return nil
}

// LoginOptBasicAuth returns a function that sets the username/password settings on login
func LoginOptBasicAuth(username string, password string) LoginOption {
	return func(operation *loginOperation) {
		operation.username = username
		operation.password = password
	}
}

// LoginOptInsecure returns a function that sets the insecure setting on login
func LoginOptInsecure(insecure bool) LoginOption {
	return func(operation *loginOperation) {
		operation.insecure = insecure
	}
}

// LoginOptTLSClientConfig returns a function that sets the TLS settings on login.
func LoginOptTLSClientConfig(certFile, keyFile, caFile string) LoginOption {
	return func(operation *loginOperation) {
		operation.certFile = certFile
		operation.keyFile = keyFile
		operation.caFile = caFile
	}
}

type (
	// LogoutOption allows specifying various settings on logout
	LogoutOption func(*logoutOperation)

	logoutOperation struct{}
)

// Logout logs out of a registry
func (c *Client) Logout(host string, opts ...LogoutOption) error {
	operation := &logoutOperation{}
	for _, opt := range opts {
		opt(operation)
	}
	if err := c.authorizer.Logout(ctx(c.out, c.debug), host); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "Removing login credentials for %s\n", host)
	return nil
}

type (
	// PullOption allows specifying various settings on pull
	PullOption func(*pullOperation)

	// PullResult is the result returned upon successful pull.
	PullResult struct {
		Manifest *DescriptorPullSummary            `json:"manifest"`
		Config   *DescriptorPullSummary            `json:"config"`
		Metadata *DescriptorPullSummary            `json:"meta"`
		Archive  *DescriptorPullSummaryWithProject `json:"archive"`
		Ref      string                            `json:"ref"`
	}

	DescriptorPullSummary struct {
		Data   []byte `json:"-"`
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	}

	DescriptorPullSummaryWithProject struct {
		DescriptorPullSummary
		Project *pkl.Project `json:"project"`
	}

	pullOperation struct {
		withPackage bool
	}
)

// Pull downloads a package from a registry
func (c *Client) Pull(ref string, options ...PullOption) (*PullResult, error) {
	parsedRef, err := parseReference(ref)
	if err != nil {
		return nil, err
	}

	operation := &pullOperation{
		withPackage: true, // By default, always download the package layer
	}
	for _, option := range options {
		option(operation)
	}

	memoryStore := content.NewMemory()
	allowedMediaTypes := []string{
		ConfigMediaType, MetadataMediaType,
	}
	minNumDescriptors := 1 // 1 for the config
	if operation.withPackage {
		minNumDescriptors++
		allowedMediaTypes = append(allowedMediaTypes, PackageLayerMediaType)
	}

	var descriptors, layers []ocispec.Descriptor
	remotesResolver, err := c.resolver(parsedRef)
	if err != nil {
		return nil, err
	}
	registryStore := content.Registry{Resolver: remotesResolver}

	manifest, err := oras.Copy(ctx(c.out, c.debug), registryStore, parsedRef.String(), memoryStore, "",
		oras.WithPullEmptyNameAllowed(),
		oras.WithAllowedMediaTypes(allowedMediaTypes),
		oras.WithLayerDescriptors(func(l []ocispec.Descriptor) {
			layers = l
		}))
	if err != nil {
		return nil, err
	}

	descriptors = append(descriptors, manifest)
	descriptors = append(descriptors, layers...)

	numDescriptors := len(descriptors)
	if numDescriptors < minNumDescriptors {
		return nil, fmt.Errorf("manifest does not contain minimum number of descriptors (%d), descriptors found: %d",
			minNumDescriptors, numDescriptors)
	}
	var configDescriptor *ocispec.Descriptor
	var pkgDescriptor *ocispec.Descriptor
	var metaDescriptor *ocispec.Descriptor

	for _, descriptor := range descriptors {
		d := descriptor
		switch d.MediaType {
		case ConfigMediaType:
			configDescriptor = &d
		case PackageLayerMediaType:
			pkgDescriptor = &d
		case MetadataMediaType:
			metaDescriptor = &d
		}
	}

	if configDescriptor == nil {
		return nil, fmt.Errorf("could not load config with mediatype %s", ConfigMediaType)
	}

	if metaDescriptor == nil {
		return nil, fmt.Errorf("could not load metadata with mediatype %s", MetadataMediaType)
	}

	if operation.withPackage && pkgDescriptor == nil {
		return nil, fmt.Errorf("manifest does not contain a layer with mediatype %s", PackageLayerMediaType)
	}

	result := &PullResult{
		Manifest: &DescriptorPullSummary{
			Digest: manifest.Digest.String(),
			Size:   manifest.Size,
		},
		Config: &DescriptorPullSummary{
			Digest: configDescriptor.Digest.String(),
			Size:   configDescriptor.Size,
		},
		Metadata: &DescriptorPullSummary{
			Digest: metaDescriptor.Digest.String(),
			Size:   metaDescriptor.Size,
		},
		Archive: &DescriptorPullSummaryWithProject{},
		Ref:     parsedRef.String(),
	}
	var getManifestErr error
	if _, manifestData, ok := memoryStore.Get(manifest); !ok {
		getManifestErr = errors.Errorf("Unable to retrieve blob with digest %s", manifest.Digest)
	} else {
		result.Manifest.Data = manifestData
	}
	if getManifestErr != nil {
		return nil, getManifestErr
	}

	var getConfigDescriptorErr error
	if _, configData, ok := memoryStore.Get(*configDescriptor); !ok {
		getConfigDescriptorErr = errors.Errorf("Unable to retrieve blob with digest %s", configDescriptor.Digest)
	} else {
		result.Config.Data = configData
		var project *pkl.Project
		if err := json.Unmarshal(configData, &project); err != nil {
			return nil, err
		}
		result.Archive.Project = project
	}
	if getConfigDescriptorErr != nil {
		return nil, getConfigDescriptorErr
	}

	var getMetadataDescriptorErr error
	if _, metadataData, ok := memoryStore.Get(*metaDescriptor); !ok {
		getMetadataDescriptorErr = errors.Errorf("Unable to retrieve blob with digest %s", metaDescriptor.Digest)
	} else {
		result.Metadata.Data = metadataData
	}
	if getMetadataDescriptorErr != nil {
		return nil, getConfigDescriptorErr
	}

	if operation.withPackage {
		var getPackageDescriptorErr error
		if _, pkgData, ok := memoryStore.Get(*pkgDescriptor); !ok {
			getPackageDescriptorErr = errors.Errorf("Unable to retrieve blob with digest %s", pkgDescriptor.Digest)
		} else {
			result.Archive.Data = pkgData
			result.Archive.Digest = pkgDescriptor.Digest.String()
			result.Archive.Size = pkgDescriptor.Size
		}
		if getPackageDescriptorErr != nil {
			return nil, getPackageDescriptorErr
		}
	}

	fmt.Fprintf(c.out, "Pulled: %s\n", result.Ref)
	fmt.Fprintf(c.out, "Digest: %s\n", result.Manifest.Digest)

	if strings.Contains(result.Ref, "_") {
		fmt.Fprintf(c.out, "%s contains an underscore.\n", result.Ref)
		fmt.Fprint(c.out, registryUnderscoreMessage+"\n")
	}

	return result, nil
}

// PullOptWithPackage returns a function that sets the withPackage setting on pull
func PullOptWithPackage(withPackage bool) PullOption {
	return func(operation *pullOperation) {
		operation.withPackage = withPackage
	}
}

type (
	// PushOption allows specifying various settings on push
	PushOption func(*pushOperation)

	// PushResult is the result returned upon successful push.
	PushResult struct {
		Manifest *descriptorPushSummary            `json:"manifest"`
		Config   *descriptorPushSummary            `json:"config"`
		Metadata *descriptorPushSummary            `json:"meta"`
		Archive  *descriptorPushSummaryWithProject `json:"archive"`
		Ref      string                            `json:"ref"`
	}

	descriptorPushSummary struct {
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	}

	descriptorPushSummaryWithProject struct {
		descriptorPushSummary
		Project *pkl.Project `json:"project"`
	}

	pushOperation struct {
		strictMode   bool
		creationTime string
	}
)

// Push uploads a package to a registry.
func (c *Client) Push(archiveFile string, metadataFile string, ref string, project *pkl.Project, options ...PushOption) (*PushResult, error) {
	parsedRef, err := parseReference(ref)
	if err != nil {
		return nil, err
	}

	operation := &pushOperation{
		strictMode: true, // By default, enable strict mode
	}
	for _, option := range options {
		option(operation)
	}

	if err != nil {
		return nil, err
	}
	if operation.strictMode {
		if !strings.HasSuffix(ref, fmt.Sprintf("/%s:%s", project.Package.Name, project.Package.Version)) {
			return nil, errors.New(
				"strict mode enabled, ref basename and tag must match the package name and version")
		}
	}
	memoryStore := content.NewMemory()

	data, err := os.ReadFile(archiveFile)
	if err != nil {
		return nil, err
	}

	metadata, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, err
	}

	pkgDescriptor, err := memoryStore.Add("", PackageLayerMediaType, data)
	if err != nil {
		return nil, err
	}

	metadataDescriptor, err := memoryStore.Add("", MetadataMediaType, metadata)
	if err != nil {
		return nil, err
	}

	configData, err := json.Marshal(project)
	if err != nil {
		return nil, err
	}

	configDescriptor, err := memoryStore.Add("", ConfigMediaType, configData)
	if err != nil {
		return nil, err
	}

	descriptors := []ocispec.Descriptor{pkgDescriptor, metadataDescriptor}

	ociAnnotations := generateOCIAnnotations(project, operation.creationTime)

	manifestData, manifest, err := content.GenerateManifest(&configDescriptor, ociAnnotations, descriptors...)
	if err != nil {
		return nil, err
	}

	if err := memoryStore.StoreManifest(parsedRef.String(), manifest, manifestData); err != nil {
		return nil, err
	}

	remotesResolver, err := c.resolver(parsedRef)
	if err != nil {
		return nil, err
	}
	registryStore := content.Registry{Resolver: remotesResolver}
	_, err = oras.Copy(ctx(c.out, c.debug), memoryStore, parsedRef.String(), registryStore, "",
		oras.WithNameValidation(nil))
	if err != nil {
		return nil, err
	}
	projectMeta := &descriptorPushSummaryWithProject{
		Project: project,
	}
	projectMeta.Digest = pkgDescriptor.Digest.String()
	projectMeta.Size = pkgDescriptor.Size
	result := &PushResult{
		Manifest: &descriptorPushSummary{
			Digest: manifest.Digest.String(),
			Size:   manifest.Size,
		},
		Config: &descriptorPushSummary{
			Digest: configDescriptor.Digest.String(),
			Size:   configDescriptor.Size,
		},
		Metadata: &descriptorPushSummary{
			Digest: metadataDescriptor.Digest.String(),
			Size:   metadataDescriptor.Size,
		},
		Archive: projectMeta,
		Ref:     parsedRef.String(),
	}

	fmt.Fprintf(c.out, "Pushed: %s\n", result.Ref)
	fmt.Fprintf(c.out, "Digest: %s\n", result.Manifest.Digest)
	if strings.Contains(parsedRef.Reference, "_") {
		fmt.Fprintf(c.out, "%s contains an underscore.\n", result.Ref)
		fmt.Fprint(c.out, registryUnderscoreMessage+"\n")
	}

	return result, err
}

// PushOptStrictMode returns a function that sets the strictMode setting on push
func PushOptStrictMode(strictMode bool) PushOption {
	return func(operation *pushOperation) {
		operation.strictMode = strictMode
	}
}

// PushOptCreationDate returns a function that sets the creation time
func PushOptCreationTime(creationTime string) PushOption {
	return func(operation *pushOperation) {
		operation.creationTime = creationTime
	}
}

// Tags provides a sorted list all semver compliant tags for a given repository
func (c *Client) Tags(ref string) ([]string, error) {
	parsedReference, err := registry.ParseReference(ref)
	if err != nil {
		return nil, err
	}

	repository := registryremote.Repository{
		Reference: parsedReference,
		Client:    c.registryAuthorizer,
		PlainHTTP: c.plainHTTP,
	}

	var registryTags []string

	registryTags, err = registry.Tags(ctx(c.out, c.debug), &repository)
	if err != nil {
		return nil, err
	}

	var tagVersions []*semver.Version
	for _, tag := range registryTags {
		// Change underscore (_) back to plus (+) for Helm
		// See https://github.com/helm/helm/issues/10166
		tagVersion, err := semver.StrictNewVersion(strings.ReplaceAll(tag, "_", "+"))
		if err == nil {
			tagVersions = append(tagVersions, tagVersion)
		}
	}

	// Sort the collection
	sort.Sort(sort.Reverse(semver.Collection(tagVersions)))

	tags := make([]string, len(tagVersions))

	for iTv, tv := range tagVersions {
		tags[iTv] = tv.String()
	}

	return tags, nil

}
