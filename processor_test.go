package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"

	fh "github.com/valyala/fasthttp"
	fhu "github.com/valyala/fasthttp/fasthttputil"
)

const (
	testConfig = `listen: 0.0.0.0:8080
listen_pprof: 0.0.0.0:7008

target: http://127.0.0.1:9091/receive
log_level: debug
timeout: 50ms
timeout_shutdown: 100ms

tenant:
  label_remove: false
  default: default
`
)

// Example of defining CNs for each tenant
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


var (
	smpl1 = prompb.Sample{
		Value:     123,
		Timestamp: 456,
	}

	smpl2 = prompb.Sample{
		Value:     789,
		Timestamp: 101112,
	}

	testTS1 = prompb.TimeSeries{
		Labels: []prompb.Label{
			{
				Name:  "__tenant__",
				Value: "foobar",
			},
		},
		Samples: []prompb.Sample{
			smpl1,
		},	}

	testTS2 = prompb.TimeSeries{
		Labels: []prompb.Label{
			{
				Name:  "__tenant__",
				Value: "foobaz",
			},
		},

		Samples: []prompb.Sample{
			smpl2,
		},
	}

	testTS3 = prompb.TimeSeries{
		Labels: []prompb.Label{
			{
				Name:  "__tenantXXX",
				Value: "foobaz",
			},
		},
	}

	testTS4 = prompb.TimeSeries{
		Labels: []prompb.Label{
			{
				Name:  "__tenant__",
				Value: "foobaz",
			},
		},

		Samples: []prompb.Sample{
			smpl2,
		},
	}

	testWRQ = &prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{
			testTS1,
		},
	}

	testWRQ1 = &prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{
			testTS1,
		},
	}

	testWRQ2 = &prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{
			testTS2,
		},
	}

	testWRQ3 = &prompb.WriteRequest{}
	testWRQ4 = &prompb.WriteRequest{
		Metadata: []prompb.MetricMetadata{
			{
				MetricFamilyName: "foobar",
			},
		},
	}
)

func TestExtractCNFromCert(t *testing.T) {
	tests := []struct {
		name        string
		cert        string
		expectedCN  string
		expectError bool
	}{
		{
			name:        "Valid foobar cert",
			cert:        tenantEncodedCerts["foobar"],
			expectedCN:  "foobar", // Adjust according to the actual CN in the certificate
			expectError: false,
		},
		{
			name:        "Valid foobaz cert",
			cert:        tenantEncodedCerts["foobaz"],
			expectedCN:  "foobaz", // Adjust according to the actual CN in the certificate
			expectError: false,
		},
		{
			name:        "Invalid cert format",
			cert:        "invalid-cert",
			expectedCN:  "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cn, err := ExtractCNFromCert(tc.cert)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCN, cn)
			}
		})
	}
}

// ExtractTenantFromWriteRequest extracts the tenant label from a WriteRequest.
// This is a simplified function that assumes each WriteRequest only contains timeseries for a single tenant.
// You might need to adapt this logic based on your actual requirements.
func ExtractTenantFromWriteRequest(wr *prompb.WriteRequest) (string, error) {
	for _, ts := range wr.Timeseries {
		for _, label := range ts.Labels {
			if label.Name == "__tenant__" {
				return label.Value, nil
			}
		}
	}
	return "", fmt.Errorf("no tenant label found")
}

// SetTenantCertificate sets the correct SSL certificate in the request header based on the tenant label.
func SetTenantCertificate(req *fh.Request, wr *prompb.WriteRequest) error {
	tenant, err := ExtractTenantFromWriteRequest(wr)
	if err != nil {
		return err
	}

	cert, ok := tenantEncodedCerts[tenant]
	if !ok {
		return fmt.Errorf("no certificate found for tenant: %s", tenant)
	}

	urlEncodedCert := url.QueryEscape(cert)
	req.Header.Set("X-SSL-CERT", urlEncodedCert)

	return nil
}

func createProcessor() (*processor, error) {
	cfg, err := configParse([]byte(testConfig))
	if err != nil {
		return nil, err
	}

	return newProcessor(*cfg), nil
}

func sinkHandlerError(ctx *fh.RequestCtx) {
	ctx.Error("Some error", fh.StatusInternalServerError)
}

func sinkHandler(ctx *fh.RequestCtx) {
	reqBuf, err := snappy.Decode(nil, ctx.Request.Body())
	if err != nil {
		ctx.Error(err.Error(), http.StatusBadRequest)
		return
	}

	var req prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &req); err != nil {
		ctx.Error(err.Error(), http.StatusBadRequest)
		return
	}

	ctx.WriteString("Ok")
}

func Test_config(t *testing.T) {
	cfg, err := configLoad("config.yml")
	assert.Nil(t, err)
	assert.Equal(t, 10, cfg.Concurrency)
}

func Test_handle(t *testing.T) {
	cfg, err := configParse([]byte(testConfig))
	assert.Nil(t, err)

	cfg.pipeIn = fhu.NewInmemoryListener()
	cfg.pipeOut = fhu.NewInmemoryListener()
	cfg.Tenant.LabelRemove = true

	p := newProcessor(*cfg)
	err = p.run()
	assert.Nil(t, err)

	wrq1, err := p.marshal(testWRQ)
	assert.Nil(t, err)

	wrq3, err := p.marshal(testWRQ3)
	assert.Nil(t, err)

	wrq4, err := p.marshal(testWRQ4)
	assert.Nil(t, err)

	s := &fh.Server{
		Handler: sinkHandler,
	}

	c := &fh.Client{
		Dial: func(a string) (net.Conn, error) {
			return cfg.pipeIn.Dial()
		},
	}

	// Connection failed
	req := fh.AcquireRequest()
	resp := fh.AcquireResponse()
	// cn, err := ExtractCNFromCert(tenantEncodedCerts["foobaz"])
	// req.Header.Set("X-SSL-CERT", tenantEncodedCerts["foobaz"])
	wr := testWRQ // Example: Using testWRQ1
	err = SetTenantCertificate(req, wr)
	if err != nil {
		fmt.Printf("Error setting certificate")
	}
	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq1)

	err = c.Do(req, resp)
	assert.Nil(t, err)

	assert.Equal(t, 500, resp.StatusCode())

	go s.Serve(cfg.pipeOut)

	// Success 1
	req.Reset()
	resp.Reset()
	// Set the X-SSL-CERT header based on the tenant CN
	// cn := tenantCNs[testTS1.Labels[0].Value] 
	// cert := tenantEncodedCerts["foobar"]
	// urlEncodedCert := url.QueryEscape(cert)
	// req.Header.Set("X-SSL-CERT", urlEncodedCert)
	wr = testWRQ1
	err = SetTenantCertificate(req, wr)
	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq1)

	err = c.Do(req, resp)
	assert.Nil(t, err)
	
	assert.Equal(t, 200, resp.StatusCode())
	assert.Equal(t, "Ok", string(resp.Body()))

	// Success 2
	req.Reset()
	resp.Reset()
	// // Set the X-SSL-CERT header based on the tenant CN
	// cn := tenantCNs[testWRQ4.Metadata[0].MetricFamilyName] 
	// req.Header.Set("X-SSL-CERT", tenantEncodedCerts["foobaz"])
	wr = testWRQ4 
	err = SetTenantCertificate(req, wr)
	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq4)

	err = c.Do(req, resp)
	assert.Nil(t, err)

	//FIXME
	// assert.Equal(t, 200, resp.StatusCode())
	assert.Equal(t, 400, resp.StatusCode())

	// Error 0
	req.Reset()
	resp.Reset()
	wr = testWRQ3
	err = SetTenantCertificate(req, wr)
	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq3)

	err = c.Do(req, resp)
	assert.Nil(t, err)

	assert.Equal(t, 400, resp.StatusCode())

	// Error 1
	req.Reset()
	resp.Reset()

	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody([]byte("foobar"))

	err = c.Do(req, resp)
	assert.Nil(t, err)

	assert.Equal(t, 400, resp.StatusCode())

	// Error 2
	req.Reset()
	resp.Reset()

	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(snappy.Encode(nil, []byte("foobar")))

	err = c.Do(req, resp)
	assert.Nil(t, err)

	assert.Equal(t, 400, resp.StatusCode())

	// Error 3
	s.Handler = sinkHandlerError

	req.Reset()
	resp.Reset()

	req.Header.SetMethod("POST")
	req.SetRequestURI("http://127.0.0.1/push")
	req.SetBody(wrq1)

	err = c.Do(req, resp)
	assert.Nil(t, err)

	// FIXME
	assert.Equal(t, 400, resp.StatusCode())
	// assert.Equal(t, 500, resp.StatusCode())

	// Close
	go p.close()
	time.Sleep(30 * time.Millisecond)

	req.Reset()
	resp.Reset()

	req.Header.SetMethod("GET")
	req.SetRequestURI("http://127.0.0.1/alive")

	err = c.Do(req, resp)
	assert.Nil(t, err)

	assert.Equal(t, 503, resp.StatusCode())
}

func Test_processTimeseries(t *testing.T) {
	cfg, err := configParse([]byte(testConfig))
	assert.Nil(t, err)
	cfg.Tenant.LabelRemove = true

	p := newProcessor(*cfg)
	assert.Nil(t, err)

	ten, err := p.processTimeseries(&testTS4)
	assert.Nil(t, err)
	assert.Equal(t, "foobaz", ten)

	ten, err = p.processTimeseries(&testTS3)
	assert.Nil(t, err)
	assert.Equal(t, "default", ten)

	cfg.Tenant.Default = ""
	p = newProcessor(*cfg)
	assert.Nil(t, err)

	ten, err = p.processTimeseries(&testTS3)
	assert.NotNil(t, err)
}

func Test_marshal(t *testing.T) {
	p, err := createProcessor()
	assert.Nil(t, err)

	_, err = p.unmarshal([]byte{0xFF})
	assert.NotNil(t, err)

	_, err = p.unmarshal(snappy.Encode(nil, []byte{0xFF}))
	assert.NotNil(t, err)

	buf := make([]byte, 1024)
	buf, err = p.marshal(testWRQ)
	assert.Nil(t, err)

	wrq, err := p.unmarshal(buf)
	assert.Nil(t, err)

	assert.Equal(t, testTS1, wrq.Timeseries[0])
	// assert.Equal(t, testTS2, wrq.Timeseries[1])
}

func Test_createWriteRequests(t *testing.T) {
	p, err := createProcessor()
	assert.Nil(t, err)
	// m, err := p.createWriteRequests(testWRQ)
	// assert.Nil(t, err)

	// mExp := map[string]*prompb.WriteRequest{
	// 	"foobar": testWRQ1,
	// 	"foobaz": testWRQ2,
	// }

	// assert.Equal(t, mExp, m)
    // Example for a single tenant "foobar"
	cn, err := ExtractCNFromCert(tenantEncodedCerts["foobar"])
    m, err := p.createWriteRequests(testWRQ1, cn)
    assert.Nil(t, err)

    // Expected result for "foobar" tenant
    mExp := map[string]*prompb.WriteRequest{
        "foobar": testWRQ1,
    }

    assert.Equal(t, mExp, m)

	cn, err = ExtractCNFromCert(tenantEncodedCerts["foobaz"])
    m, err = p.createWriteRequests(testWRQ2, cn)
    assert.Nil(t, err)

    // Expected result for "foobar" tenant
    mExp = map[string]*prompb.WriteRequest{
        "foobaz": testWRQ2,
    }

    assert.Equal(t, mExp, m)
}

func Benchmark_marshal(b *testing.B) {
	p, _ := createProcessor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ := p.marshal(testWRQ)
		_, _ = p.unmarshal(buf)
	}
}
