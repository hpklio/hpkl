package vals

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/helmfile/vals/pkg/api"
	"github.com/helmfile/vals/pkg/config"
	"github.com/helmfile/vals/pkg/log"
	"github.com/helmfile/vals/pkg/providers/awskms"
	"github.com/helmfile/vals/pkg/providers/awssecrets"
	"github.com/helmfile/vals/pkg/providers/azurekeyvault"
	"github.com/helmfile/vals/pkg/providers/bitwarden"
	"github.com/helmfile/vals/pkg/providers/conjur"
	"github.com/helmfile/vals/pkg/providers/doppler"
	"github.com/helmfile/vals/pkg/providers/echo"
	"github.com/helmfile/vals/pkg/providers/envsubst"
	"github.com/helmfile/vals/pkg/providers/file"
	"github.com/helmfile/vals/pkg/providers/gcpsecrets"
	"github.com/helmfile/vals/pkg/providers/gcs"
	"github.com/helmfile/vals/pkg/providers/gitlab"
	"github.com/helmfile/vals/pkg/providers/gkms"
	"github.com/helmfile/vals/pkg/providers/googlesheets"
	"github.com/helmfile/vals/pkg/providers/hcpvaultsecrets"
	"github.com/helmfile/vals/pkg/providers/httpjson"
	"github.com/helmfile/vals/pkg/providers/k8s"
	"github.com/helmfile/vals/pkg/providers/onepasswordconnect"
	"github.com/helmfile/vals/pkg/providers/pulumi"
	"github.com/helmfile/vals/pkg/providers/s3"
	"github.com/helmfile/vals/pkg/providers/sops"
	"github.com/helmfile/vals/pkg/providers/ssm"
	"github.com/helmfile/vals/pkg/providers/tfstate"
	"github.com/helmfile/vals/pkg/providers/vault"
	"gopkg.in/yaml.v2"
)

const (
	TypeMap    = "map"
	TypeString = "string"

	FormatRaw  = "raw"
	FormatYAML = "yaml"

	KeyProvider   = "provider"
	KeyName       = "name"
	KeyKeys       = "keys"
	KeyPaths      = "paths"
	KeyType       = "type"
	KeyFormat     = "format"
	KeyInline     = "inline"
	KeyPrefix     = "prefix"
	KeyPath       = "path"
	KeySetForKey  = "setForKeys"
	KeySet        = "set"
	KeyValuesFrom = "valuesFrom"

	// secret cache size
	defaultCacheSize = 512

	ProviderVault              = "vault"
	ProviderS3                 = "s3"
	ProviderGCS                = "gcs"
	ProviderGitLab             = "gitlab"
	ProviderSSM                = "awsssm"
	ProviderKms                = "awskms"
	ProviderSecretsManager     = "awssecrets"
	ProviderSOPS               = "sops"
	ProviderEcho               = "echo"
	ProviderFile               = "file"
	ProviderGCPSecretManager   = "gcpsecrets"
	ProviderGoogleSheets       = "googlesheets"
	ProviderTFState            = "tfstate"
	ProviderTFStateGS          = "tfstategs"
	ProviderTFStateS3          = "tfstates3"
	ProviderTFStateAzureRM     = "tfstateazurerm"
	ProviderTFStateRemote      = "tfstateremote"
	ProviderAzureKeyVault      = "azurekeyvault"
	ProviderEnvSubst           = "envsubst"
	ProviderOnePasswordConnect = "onepasswordconnect"
	ProviderDoppler            = "doppler"
	ProviderPulumiStateAPI     = "pulumistateapi"
	ProviderGKMS               = "gkms"
	ProviderK8s                = "k8s"
	ProviderConjur             = "conjur"
	ProviderHCPVaultSecrets    = "hcpvaultsecrets"
	ProviderHttpJsonManager    = "httpjson"
	ProviderBitwarden          = "bw"
)

var (
	EnvFallbackPrefix = "VALS_"
)

type KeysRuntime struct {
	providers map[string]api.Provider

	Options Options

	logger *log.Logger

	m sync.Mutex
}

type Getter struct {
	GetDoc func(key string) (map[string]interface{}, error)
}

type Options struct {
	LogOutput             io.Writer
	CacheSize             int
	ExcludeSecret         bool
	FailOnMissingKeyInMap bool
}

func New(opts Options) (*KeysRuntime, error) {
	r := &KeysRuntime{
		providers: map[string]api.Provider{},
		Options:   opts,
		logger: log.New(log.Config{
			Output: opts.LogOutput,
		}),
	}
	return r, nil
}

// nolint
func (r *KeysRuntime) prepare() (*Getter, error) {
	var err error

	uriToProviderHash := func(uri *url.URL) string {
		bs := []byte{}
		bs = append(bs, []byte(uri.Scheme)...)
		query := uri.Query().Encode()
		bs = append(bs, []byte(query)...)
		return fmt.Sprintf("%x", md5.Sum(bs))
	}

	createProvider := func(scheme string, uri *url.URL) (api.Provider, error) {
		scheme = strings.TrimPrefix(scheme, "ref+")
		query := uri.Query()

		m := map[string]interface{}{}
		for key, params := range query {
			if len(params) > 0 {
				m[key] = params[0]
			}
		}

		envFallback := func(k string) string {
			key := fmt.Sprintf("%s%s", EnvFallbackPrefix, strings.ToUpper(k))
			return os.Getenv(key)
		}

		conf := config.MapConfig{M: m, FallbackFunc: envFallback}

		switch scheme {
		case ProviderVault:
			p := vault.New(r.logger, conf)
			return p, nil
		case ProviderS3:
			// ref+s3://foo/bar?region=ap-northeast-1#/baz
			// 1. GetObject for the bucket foo and key bar
			// 2. Then extracts the value for key baz(=/foo/bar/baz) from the result from step 1.
			p := s3.New(r.logger, conf)
			return p, nil
		case ProviderGCS:
			// vals+gcs://foo/bar?generation=timestamp#/baz
			// 1. GetObject for the bucket foo and key bar
			// 2. Then extracts the value for key baz(=/foo/bar/baz) from the result from step 1.
			p := gcs.New(conf)
			return p, nil
		case ProviderGitLab:
			// vals+gitlab://project/variable#key
			p := gitlab.New(conf)
			return p, nil
		case ProviderSSM:
			// ref+awsssm://foo/bar?region=ap-northeast-1#/baz
			// 1. GetParametersByPath for the prefix /foo/bar
			// 2. Then extracts the value for key baz(=/foo/bar/baz) from the result from step 1.
			p := ssm.New(r.logger, conf)
			return p, nil
		case ProviderSecretsManager:
			// ref+awssecrets://foo/bar?region=ap-northeast-1#/baz
			// 1. Get secret for key foo/bar, parse it as yaml
			// 2. Then extracts the value for key baz) from the result from step 1.
			p := awssecrets.New(r.logger, conf)
			return p, nil
		case ProviderSOPS:
			p := sops.New(r.logger, conf)
			return p, nil
		case ProviderEcho:
			p := echo.New(conf)
			return p, nil
		case ProviderFile:
			p := file.New(conf)
			return p, nil
		case ProviderGCPSecretManager:
			p := gcpsecrets.New(conf)
			return p, nil
		case ProviderGoogleSheets:
			return googlesheets.New(conf), nil
		case ProviderTFState:
			p := tfstate.New(conf, "")
			return p, nil
		case ProviderTFStateGS:
			p := tfstate.New(conf, "gs")
			return p, nil
		case ProviderTFStateS3:
			p := tfstate.New(conf, "s3")
			return p, nil
		case ProviderTFStateAzureRM:
			p := tfstate.New(conf, "azurerm")
			return p, nil
		case ProviderTFStateRemote:
			p := tfstate.New(conf, "remote")
			return p, nil
		case ProviderAzureKeyVault:
			p := azurekeyvault.New(conf)
			return p, nil
		case ProviderKms:
			p := awskms.New(conf)
			return p, nil
		case ProviderEnvSubst:
			p := envsubst.New(conf)
			return p, nil
		case ProviderOnePasswordConnect:
			p := onepasswordconnect.New(conf)
			return p, nil
		case ProviderDoppler:
			p := doppler.New(r.logger, conf)
			return p, nil
		case ProviderPulumiStateAPI:
			p := pulumi.New(r.logger, conf, "pulumistateapi")
			return p, nil
		case ProviderGKMS:
			p := gkms.New(r.logger, conf)
			return p, nil
		case ProviderK8s:
			return k8s.New(r.logger, conf)
		case ProviderConjur:
			p := conjur.New(r.logger, conf)
			return p, nil
		case ProviderHCPVaultSecrets:
			p := hcpvaultsecrets.New(r.logger, conf)
			return p, nil
		case ProviderHttpJsonManager:
			p := httpjson.New(r.logger, conf)
			return p, nil
		case ProviderBitwarden:
			p := bitwarden.New(r.logger, conf)
			return p, nil
		}
		return nil, fmt.Errorf("no provider registered for scheme %q", scheme)
	}

	updateProviders := func(uri *url.URL, hash string) (api.Provider, error) {
		r.m.Lock()
		defer r.m.Unlock()
		p, ok := r.providers[hash]
		if !ok {
			var scheme string
			scheme = uri.Scheme
			scheme = strings.Split(scheme, "://")[0]

			p, err = createProvider(scheme, uri)
			if err != nil {
				return nil, err
			}

			r.providers[hash] = p
		}
		return p, nil
	}

	getter := Getter{
		GetDoc: func(key string) (map[string]interface{}, error) {
			uri, err := url.Parse(key)
			if err != nil {
				return nil, err
			}

			hash := uriToProviderHash(uri)

			p, err := updateProviders(uri, hash)

			if err != nil {
				return nil, err
			}

			var frag string
			frag = uri.Fragment
			frag = strings.TrimPrefix(frag, "#")
			frag = strings.TrimPrefix(frag, "/")

			var components []string
			var host string

			{
				host = uri.Host

				if host != "" {
					components = append(components, host)
				}
			}

			{
				path2 := uri.Path
				path2 = strings.TrimPrefix(path2, "#")
				if host != "" {
					path2 = strings.TrimPrefix(path2, "/")
				}

				if path2 != "" {
					components = append(components, path2)
				}
			}

			path := strings.Join(components, "/")

			return p.GetStringMap(path)
		},
	}
	return &getter, nil
}

func (r *KeysRuntime) GetString(key string) (string, error) {
	getter, err := r.prepare()
	if err != nil {
		return "", err
	}

	return getter.GetString(key)
}

func (r *KeysRuntime) GetMap(key string) (map[string]interface{}, error) {
	getter, err := r.prepare()
	if err != nil {
		return nil, err
	}

	return getter.GetMap(key)
}

func (g *Getter) findElement(doc map[string]interface{}, key string) (interface{}, error) {
	uri, err := url.Parse(key)
	if err != nil {
		return "", err
	}

	var frag string
	frag = uri.Fragment
	frag = strings.TrimPrefix(frag, "#")
	frag = strings.TrimPrefix(frag, "/")

	if len(frag) > 0 {
		keys := strings.Split(frag, "/")
		obj := doc
		for i, k := range keys {
			newobj := map[string]interface{}{}
			switch t := obj[k].(type) {
			case string:
				if i != len(keys)-1 {
					return "", fmt.Errorf("unexpected type of value for key at %d=%s in %v: expected map[string]interface{}, got %v(%T)", i, k, keys, t, t)
				}
				return t, nil
			case map[string]interface{}:
				newobj = t
			case map[interface{}]interface{}:
				for k, v := range t {
					newobj[fmt.Sprintf("%v", k)] = v
				}
			case []interface{}:
				for k, v := range t {
					newobj[fmt.Sprintf("%v", k)] = v
				}
			}
			obj = newobj
		}
		return obj, nil
	} else {
		return doc, nil
	}

}

func (g *Getter) GetString(key string) (string, error) {
	doc, err := g.GetDoc(key)
	if err != nil {
		return "", err
	}

	el, err := g.findElement(doc, key)
	if err != nil {
		return "", err
	}

	switch t := el.(type) {
	case string:
		return t, nil
	default:
		return "", fmt.Errorf("unexpected type of value for key at %s in %v: expected string, got %v(%T)", key, doc, t, t)

	}
}

func (g *Getter) GetMap(key string) (map[string]interface{}, error) {
	doc, err := g.GetDoc(key)
	if err != nil {
		return nil, err
	}

	el, err := g.findElement(doc, key)
	if err != nil {
		return nil, err
	}

	switch t := el.(type) {
	case map[string]interface{}:
		return t, nil
	case []interface{}:
		newobj := map[string]interface{}{}
		for k, v := range t {
			newobj[fmt.Sprintf("%v", k)] = v
		}
		return newobj, nil
	default:
		return nil, fmt.Errorf("unexpected type of value for key at %s in %v: expected map[string]interface{}, got %v(%T)", key, doc, t, t)

	}
}

func cloneMap(m map[string]interface{}) map[string]interface{} {
	bs, err := yaml.Marshal(m)
	if err != nil {
		panic(err)
	}
	out := map[string]interface{}{}
	if err := yaml.Unmarshal(bs, &out); err != nil {
		panic(err)
	}
	return out
}
