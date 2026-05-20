package main

import (
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	fh "github.com/valyala/fasthttp"
	fhu "github.com/valyala/fasthttp/fasthttputil"
)

// CN-validation is an optional, opt-in feature added by the RunWhen
// downstream fork (not in upstream blind-oracle/cortex-tenant). Tests
// for it live in their own file so the upstream test fixtures
// (testTS1/testTS2/testWRQ/...) in processor_test.go stay verbatim and
// continue to regression-guard the upstream-equivalent disabled path.
//
// All tests below set `cn_validation.enabled: true`. The two test
// certificates are self-signed PEMs with `Subject CN=foobar` and
// `Subject CN=foobaz` so they line up with the tenant values already
// present in testTS1/testTS2.

const testConfigCN = `listen: 0.0.0.0:8080
listen_pprof: 0.0.0.0:7008

target: http://127.0.0.1:9091/receive
log_level: debug
timeout: 50ms
timeout_shutdown: 100ms

tenant:
  label_remove: false
  default: default

cn_validation:
  enabled: true
  header: X-SSL-CERT
`

// Self-signed test certs. CN equals the map key. Generated with:
//
//	openssl req -x509 -newkey rsa:4096 -days 9999 -nodes -keyout /dev/null \
//	  -subj "/CN=<tenant>" -out cert.pem
//
// No private keys are stored; the public-cert PEMs are safe to commit
// and make the CN-extraction tests hermetic (no fixture files, no test-
// time cert generation).
var tenantEncodedCerts = map[string]string{
	"foobar": `-----BEGIN CERTIFICATE-----
MIIFBTCCAu2gAwIBAgIUdnKIOQYa56CmlK/KNcxycMobmpswDQYJKoZIhvcNAQEL
BQAwETEPMA0GA1UEAwwGZm9vYmFyMCAXDTI0MDIyMDIzMTMyNVoYDzIwNTEwNzA4
MjMxMzI1WjARMQ8wDQYDVQQDDAZmb29iYXIwggIiMA0GCSqGSIb3DQEBAQUAA4IC
DwAwggIKAoICAQC/zeC8CD0+YrmetDIAL+8Qy43Hd2rU0G4Dk3+yRuvSDJxz8xHe
QVSFo4KGLJdB7nf6Y5hBX7PfpdQDO7aH1xnx6lzr7xf+y7y8EKgjG4bySrogmP31
CI5YghVYeLo55Lx6zXpOtfau3aaB9RuCret3qAuWoBe4dLeWfaYNocEiaOwdnxEm
i7K/q/udeZ9cpaDWcpU+d+ThbmFDod2biiZbw8/1qAfjGGWFt8hoLjND+mfv/nS0
0ZiIxsJ6f/8GZ3oO5uDesY1gZO29XPrS14mDt21h4JdlJfmstWFAWnKho/FsyIlv
GZ1mJ4uw+zVsK9/Q+2UB6YdVCAzbMHyHMbjgwYsIDYYmKarJtfpgn58lMWh1g7i4
SAEIzKHw/dQVAB98tNrJWq/OWyeAXinndW3u2U87coTV70zrQRv8N80t6r8yi24B
Wd8XIDXlt2sTOvLmpzmgPQdOAnsBpvvtHatrllRhLfctg5x6Z39bD6eWu669ioJA
ijPTh3fK3vdMpa3Oi3ZCBFvpjIdr3nWtKS4GiZOak0vS9AcTiWiXLXHUs53lz6IY
vEE88cXXOqBpYMCTDscFV1OsHx9OCBmNTSo9qhW3vPxwHfaz0X5ezfCsDchxJd30
JrQJVkh/acs3P0KL0p+HVIORMOEhgSijh6o0l5AnEoFTU3frk1vCJ0g95QIDAQAB
o1MwUTAdBgNVHQ4EFgQUPqGFnVJ8n8aikX+2E8nnLU/GdEowHwYDVR0jBBgwFoAU
PqGFnVJ8n8aikX+2E8nnLU/GdEowDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0B
AQsFAAOCAgEAvVfzm+atTjg0a2lFixTubWRmbsFKaj8SQCCfmrerE1ExR+445Rhi
E9s1cJRa/Bcelg7RaawsYLt4XkSU4nuW0uoQNTAuyYaRi0lu8bEpID5ObyihiS6c
7yo6U/rg2i6NFQmkDX4HoDPieKtkQrdv3B98SaUd58fG2CsVBuXusQdy9bmW1hJN
FF55aZ7AsvxqS2l4YpA1yFW71t1xMe/wY9hiev9bRwLWi0Tt+tjFmJkqRDwN0sHa
hfIWLN9KHVD1kOLZ4bPNoS7CfA05929L13C8Ty2Ow+bfT6IIquLPYOSnsamjw2Ez
zrRa5vWRA1JJIkS9MbQxQAoPb76LMbd5IY4b5RqnvsYUPyEuxs+/FGKXpPPdBKxx
aCipQC2uKIB2aF+sunjg3qwyjijpZREPDjO1AD2rkqlbaT6lFuv/tc9Q2NyXVEkL
VwpKTccB5kxxhVkLN6P1YZtXr9l+aAA96nvAomoz+ycZ5SF688sP8NYROQ8MpKDU
2BFiaRP2PMZIKuLDIFXHROETVP44uF4x4ZhtZOeBsQeRgAPTE3mg9wUpg6yckyZI
DvDAGddY8y597WJZhe1dj5V3hRQ58UqhxCmNnuAdC9H8l/oDVM38EYGl0JWXCnXe
Gk+JlC42pOuf1ZWYtW3FfGqdWOYKiH60CqAiz2KojenSDjC/3hC0vc8=
-----END CERTIFICATE-----`,
	"foobaz": `-----BEGIN CERTIFICATE-----
MIIFBTCCAu2gAwIBAgIUNkdE/Xq0uajd3NTR3YwwgHF+JJcwDQYJKoZIhvcNAQEL
BQAwETEPMA0GA1UEAwwGZm9vYmF6MCAXDTI0MDIyMDIzMTMxNloYDzIwNTEwNzA4
MjMxMzE2WjARMQ8wDQYDVQQDDAZmb29iYXowggIiMA0GCSqGSIb3DQEBAQUAA4IC
DwAwggIKAoICAQDaUnt5NDukaf3xsHYkslA6EfAhrY5eIELSMWC0Gba946M8YICH
kALd9uxoecNr4lxN+KaNNg7GQVw8fQdI2KLL5As5ZPfkh4qdL7cCPCP0tH2i/CU1
wcMRFGkUyHSkxUS43MnX9zu9nS1UpTz0+l1In8Z2mHcYIcCEeeF4l5PDsEKzNe5R
gFMc58AJxZan6XL5527oMmmKNa5+zm8NMNtAc9KvLRI/3/U1iryGAjLse86cBo5V
OnYqN4EpSBbVaqRxcXuJY1dmUycWm8V2GyFOmK2xrKUiBdpcrwxJ3KHk1ggf+EyH
gwx1dDLYV7fZ+xWCAP5D/JaBTGhKt+h5JpIJESsruj4aQoREVQBB8QlLdKjx+xYT
Lj4alP5A7CKcYkCejc/u6JyLMM3nuRFVatoyGc4UKB9a9uxm6PZYx6HXxlZAypK8
DdkN7cp/Yp9YYVerSDay8U9iIGH6Q716MAi9B9xAgwGjQWlTmitc4ioTPE8i3KUd
3vSE78teUwZPkEGiyjAlAtJaO9Ll5shLzjl2fp7nfkHdHPw6z1J2HgCZhZ4oGoUc
GZYHJxNvUJGQFhMb5VEQ8jCEGQuFNMd743r7FjA6KX99SyddTDxhawlh5DAwl1st
yBSP7oDLlNX3pjtLGTAYt9ycvaGS2OgcWWU8eo+h+E/HXvNjaMOsntkcSwIDAQAB
o1MwUTAdBgNVHQ4EFgQUREQ0Beayu+HaHGvK23tsQ4zZF/8wHwYDVR0jBBgwFoAU
REQ0Beayu+HaHGvK23tsQ4zZF/8wDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0B
AQsFAAOCAgEApgO3rCVyn+MXDDh7YEZlyxx4EWLQDYZV56GmmUXfx8uyVZOb/NVq
h9G77v2ZfLiiiHRmvXl+pziRM+TpiYTVu8vIDqRNthi+UtLa+loZFd38E/Si+EIZ
UM+wLDWDay9gH/tIuUqzy49r5k0gzlpArveaB8dNIKa0sPhXzT1ni9Kbb36iGFNQ
6nPqN7RdEuIAdqROuVLXHMgF6vfU/t03YnFQSubGyRnNLIEY1qvY/yLxRRRJSEdp
eKpeOl392reAHBeMcnXpCpm9Hv778lKmErC+gx9OX//j0cu9dd4Erk/VNC666TLR
9n4YpcFb3CFaB7gctZCb8eLO2LSqmJDq0UIPXODl2JFX+4/A222s5FXNakAk0tc/
tu3IJ4CSzp0lyzPTHkg02Vybn2x4iWA+6TGpjApnj8AL/OQPgASEnGwsf1TOmq6+
+OYDVk23KGixT990qurF+h9jDOZwYNmn/3tmCFTF5cSRaS5LalRTn0Lvng1Uf9S7
gLT1BBPb08ez9oyWJg3awBaSvo/kXrvM1moBFc8rAB3rOPrVKpDd5D+8fjrw/O/N
1ZnzklxFo/2g/3oCQ+d4L68ZZSphGDbncw+o5lA7qn3VhvbJHQHKsDAw8V7MfkZw
iPNb0G+ABcCSDVFIRtvNYfqziR0NzGvbQNM0A/cCgCDATLEpTUP9omo=
-----END CERTIFICATE-----`,
}

func newCNProcessor(t *testing.T) *processor {
	t.Helper()
	cfg, err := configParse([]byte(testConfigCN))
	assert.Nil(t, err)
	cfg.pipeIn = fhu.NewInmemoryListener()
	cfg.pipeOut = fhu.NewInmemoryListener()
	return newProcessor(*cfg)
}

func TestExtractCNFromCert(t *testing.T) {
	tests := []struct {
		name        string
		cert        string
		expectedCN  string
		expectError bool
	}{
		{"valid PEM CN=foobar", tenantEncodedCerts["foobar"], "foobar", false},
		{"valid PEM CN=foobaz", tenantEncodedCerts["foobaz"], "foobaz", false},
		{"URL-encoded PEM", url.QueryEscape(tenantEncodedCerts["foobar"]), "foobar", false},
		{"garbage input", "not-a-cert", "", true},
		{"empty string", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cn, err := ExtractCNFromCert(tc.cert)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCN, cn)
		})
	}
}

func Test_createWriteRequests_CNValidation_Match(t *testing.T) {
	p := newCNProcessor(t)

	// testWRQ1's only timeseries has __tenant__=foobar, matching CN.
	m, err := p.createWriteRequests(testWRQ1, "foobar")
	assert.Nil(t, err)
	mExp := map[string]*prompb.WriteRequest{"foobar": testWRQ1}
	assert.Equal(t, mExp, m)
}

func Test_createWriteRequests_CNValidation_Mismatch(t *testing.T) {
	p := newCNProcessor(t)

	// CN says "wrong-cn" but tenant label says "foobar" → reject.
	_, err := p.createWriteRequests(testWRQ1, "wrong-cn")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match certificate CN")
}

func Test_createWriteRequests_CNValidation_MixedTenantsRejected(t *testing.T) {
	p := newCNProcessor(t)

	// testWRQ has two tenants (foobar + foobaz); a single CN cannot
	// authorize both, so the request must be rejected at the first
	// mismatch.
	_, err := p.createWriteRequests(testWRQ, "foobar")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match certificate CN")
}

func Test_handle_CNValidation_MissingHeader(t *testing.T) {
	p := newCNProcessor(t)
	assert.Nil(t, p.run())

	// Stand up a sink so /push has somewhere to forward to once we get
	// past the CN gate — irrelevant for this test (we expect 400 before
	// dispatch), but newProcessor's pipeOut listener still has to be
	// served or the test client hangs on close.
	go (&fh.Server{Handler: func(ctx *fh.RequestCtx) { ctx.WriteString("Ok") }}).Serve(p.cfg.pipeOut)
	defer p.close()

	wrq1, err := p.marshal(testWRQ1)
	assert.Nil(t, err)

	c := &fh.Client{
		Dial: func(_ string) (net.Conn, error) { return p.cfg.pipeIn.Dial() },
	}

	req := fh.AcquireRequest()
	resp := fh.AcquireResponse()
	defer fh.ReleaseRequest(req)
	defer fh.ReleaseResponse(resp)

	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq1)

	err = c.Do(req, resp)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
	assert.Contains(t, string(resp.Body()), "X-SSL-CERT")
}

func Test_handle_CNValidation_HeaderPresent_MatchesTenant(t *testing.T) {
	p := newCNProcessor(t)
	assert.Nil(t, p.run())

	go (&fh.Server{Handler: func(ctx *fh.RequestCtx) { ctx.WriteString("Ok") }}).Serve(p.cfg.pipeOut)
	defer p.close()

	// Build the wire body from testWRQ1 (single timeseries, tenant=foobar).
	wrq1, err := p.marshal(testWRQ1)
	assert.Nil(t, err)

	c := &fh.Client{
		Dial: func(_ string) (net.Conn, error) { return p.cfg.pipeIn.Dial() },
	}

	req := fh.AcquireRequest()
	resp := fh.AcquireResponse()
	defer fh.ReleaseRequest(req)
	defer fh.ReleaseResponse(resp)

	// Url-encode the PEM the way an nginx `$ssl_client_escaped_cert`
	// variable would — exercises ExtractCNFromCert's fallback path.
	req.Header.SetMethod("POST")
	req.Header.Set("X-SSL-CERT", url.QueryEscape(tenantEncodedCerts["foobar"]))
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq1)

	err = c.Do(req, resp)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "Ok", string(resp.Body()))

	// Be polite to the sink server before close
	time.Sleep(10 * time.Millisecond)
}

func Test_handle_CNValidation_HeaderPresent_WrongTenant(t *testing.T) {
	p := newCNProcessor(t)
	assert.Nil(t, p.run())

	go (&fh.Server{Handler: func(ctx *fh.RequestCtx) { ctx.WriteString("Ok") }}).Serve(p.cfg.pipeOut)
	defer p.close()

	// testWRQ1 has tenant=foobar, but we send a foobaz cert → 400.
	wrq1, err := p.marshal(testWRQ1)
	assert.Nil(t, err)

	c := &fh.Client{
		Dial: func(_ string) (net.Conn, error) { return p.cfg.pipeIn.Dial() },
	}

	req := fh.AcquireRequest()
	resp := fh.AcquireResponse()
	defer fh.ReleaseRequest(req)
	defer fh.ReleaseResponse(resp)

	req.Header.SetMethod("POST")
	// PEMs contain newlines, which fasthttp does not preserve in
	// header values; mirror the matching-test's URL-encoding so the
	// gate sees a real cert and reaches the CN-vs-tenant comparison
	// (rather than failing earlier at the PEM-decode step).
	req.Header.Set("X-SSL-CERT", url.QueryEscape(tenantEncodedCerts["foobaz"]))
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq1)

	err = c.Do(req, resp)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
	assert.Contains(t, string(resp.Body()), "does not match certificate CN")
}

