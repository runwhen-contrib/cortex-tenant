package main

import (
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	fhu "github.com/valyala/fasthttp/fasthttputil"
	"gopkg.in/yaml.v2"
)

type config struct {
	Listen      string
	ListenPprof string `yaml:"listen_pprof"`

	Target string

	LogLevel        string `yaml:"log_level"`
	Timeout         time.Duration
	TimeoutShutdown time.Duration `yaml:"timeout_shutdown"`
	Concurrency     int

	Tenant struct {
		Label       string
		LabelRemove bool `yaml:"label_remove"`
		Header      string
		Default     string
		AcceptAll   bool `yaml:"accept_all"`
	}

	// CNValidation gates an optional mTLS-CN-equals-tenant check. When
	// Enabled is false (the default), behavior is identical to the
	// upstream blind-oracle/cortex-tenant binary: the request's
	// `__tenant__` (or configured) label is trusted as-is and forwarded
	// to Cortex/Mimir in the configured tenant header.
	//
	// When Enabled is true, the proxy reads a PEM-encoded client cert
	// from the Header below (typically injected by an upstream nginx/
	// envoy doing mTLS termination), extracts the Subject Common Name
	// (CN), and requires every timeseries in the write request to carry
	// a tenant label equal to that CN. This prevents a client with a
	// cert for tenant `A` from writing samples labelled for tenant `B`.
	//
	// Recommended deployment: enable when running behind an mTLS-
	// terminating ingress that injects the client cert into a header
	// (e.g. nginx `ssl_client_escaped_cert` in `X-SSL-CERT`); leave
	// disabled when running in a trusted internal network where the
	// tenant label is set by a trusted producer.
	CNValidation struct {
		Enabled bool   `yaml:"enabled"`
		Header  string `yaml:"header"`
	} `yaml:"cn_validation"`

	pipeIn  *fhu.InmemoryListener
	pipeOut *fhu.InmemoryListener
}

func configParse(b []byte) (*config, error) {
	cfg := &config{}
	if err := yaml.UnmarshalStrict(b, cfg); err != nil {
		return nil, errors.Wrap(err, "Unable to parse config")
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	if cfg.Concurrency == 0 {
		cfg.Concurrency = 512
	}

	if cfg.Tenant.Header == "" {
		cfg.Tenant.Header = "X-Scope-OrgID"
	}

	if cfg.Tenant.Label == "" {
		cfg.Tenant.Label = "__tenant__"
	}

	if cfg.CNValidation.Header == "" {
		cfg.CNValidation.Header = "X-SSL-CERT"
	}

	return cfg, nil
}

func configLoad(file string) (*config, error) {
	y, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read config")
	}

	return configParse(y)
}
