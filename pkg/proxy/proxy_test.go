package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/webhook"
)

var (
	rtr    *mux.Router
	server *httptest.Server
)

func setup() {
	rtr = mux.NewRouter()
	server = httptest.NewServer(rtr)

	os.Setenv(webhook.AzureAuthorityHostEnvVar, "https://login.microsoftonline.com/")
	os.Setenv(webhook.AzureTenantIDEnvVar, "tenant_id")
}

func teardown() {
	server.Close()

	os.Unsetenv(webhook.AzureAuthorityHostEnvVar)
	os.Unsetenv(webhook.AzureTenantIDEnvVar)
}

func TestProxy_MSIHandler(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "client_id is missing",
			path:               `/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fvault.azure.net%2F`,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "The client_id parameter or AZURE_CLIENT_ID environment variable must be set\n",
		},
		{
			name:               "resource is missing",
			path:               `/metadata/identity/oauth2/token?api-version=2018-02-01&client_id=client_id`,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "The resource parameter is required.\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setup()
			defer teardown()

			p := &proxy{logger: mlog.New()}
			rtr.PathPrefix(tokenPathPrefix).HandlerFunc(p.msiHandler)
			rtr.PathPrefix("/").HandlerFunc(p.defaultPathHandler)

			req, err := http.NewRequest(http.MethodGet, test.path, nil)
			if err != nil {
				t.Error(err)
			}

			recorder := httptest.NewRecorder()
			rtr.ServeHTTP(recorder, req)

			if recorder.Code != test.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", test.expectedStatusCode, recorder.Code)
			}
			if recorder.Body.String() != test.expectedBody {
				t.Errorf("expected body %s, got %s", test.expectedBody, recorder.Body.String())
			}
		})
	}
}

func TestRouterPathPrefix(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedBody string
	}{
		{
			name:         "token request",
			path:         "/metadata/identity/oauth2/token/",
			expectedBody: "token_request_handler",
		},
		{
			name:         "token request without / suffix",
			path:         "/metadata/identity/oauth2/token",
			expectedBody: "token_request_handler",
		},
		{
			name:         "token request with upper case metadata",
			path:         "/Metadata/identity/oauth2/token/",
			expectedBody: "token_request_handler",
		},
		{
			name:         "token request with upper case identity",
			path:         "/metadata/Identity/oauth2/token/",
			expectedBody: "default_handler",
		},
		{
			name:         "instance metadata request",
			path:         "/metadata/instance",
			expectedBody: "default_handler",
		},
		{
			name:         "instance metadata request with upper case metadata",
			path:         "/Metadata/instance",
			expectedBody: "default_handler",
		},
		{
			name:         "instance metadata request / suffix",
			path:         "/Metadata/instance/",
			expectedBody: "default_handler",
		},
		{
			name:         "default metadata request",
			path:         "/metadata/",
			expectedBody: "default_handler",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setup()
			defer teardown()

			rtr.PathPrefix(tokenPathPrefix).HandlerFunc(testTokenHandler)
			rtr.PathPrefix("/").HandlerFunc(testDefaultHandler)

			req, err := http.NewRequest(http.MethodGet, server.URL+test.path, nil)
			if err != nil {
				t.Error(err)
			}

			recorder := httptest.NewRecorder()
			rtr.ServeHTTP(recorder, req)
			if recorder.Body.String() != test.expectedBody {
				t.Errorf("Expected body %s, got %s", test.expectedBody, recorder.Body.String())
			}
		})
	}
}

func TestParseTokenRequest(t *testing.T) {
	tests := []struct {
		name             string
		req              *http.Request
		expectedClientID string
		expectedResource string
	}{
		{
			name:             "no query params",
			req:              &http.Request{URL: &url.URL{Path: "/metadata/identity/oauth2/token/"}},
			expectedClientID: "",
			expectedResource: "",
		},
		{
			name: "client_id query param set",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "client_id=client_id",
				},
			},
			expectedClientID: "client_id",
			expectedResource: "",
		},
		{
			name: "resource query param set",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "resource=resource",
				},
			},
			expectedClientID: "",
			expectedResource: "resource",
		},
		{
			name: "client_id query param set and resource query param set",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "client_id=client_id&resource=resource",
				},
			},
			expectedClientID: "client_id",
			expectedResource: "resource",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := test.req
			clientID, resource := parseTokenRequest(req)
			if clientID != test.expectedClientID {
				t.Errorf("expected clientID %s, got %s", test.expectedClientID, clientID)
			}
			if resource != test.expectedResource {
				t.Errorf("expected resource %s, got %s", test.expectedResource, resource)
			}
		})
	}
}

func TestReadJWTFromFS(t *testing.T) {
	tests := []struct {
		name          string
		writeFile     func() string
		expectedToken string
		expectedError bool
	}{
		{
			name: "valid token",
			writeFile: func() string {
				tokenFilePath := filepath.Join(os.TempDir(), "test-token")
				if err := os.WriteFile(tokenFilePath, []byte("token"), 0600); err != nil {
					t.Error(err)
				}
				return tokenFilePath
			},
			expectedToken: "token",
			expectedError: false,
		},
		{
			name: "no token",
			writeFile: func() string {
				return filepath.Join(os.TempDir(), "test-token-0")
			},
			expectedToken: "",
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokenFilePath := test.writeFile()
			defer os.Remove(tokenFilePath)

			token, err := readJWTFromFS(tokenFilePath)
			if err != nil && !test.expectedError {
				t.Error(err)
			}
			if err == nil && test.expectedError {
				t.Error("expected error, got none")
			}
			if token != test.expectedToken {
				t.Errorf("expected token %s, got %s", test.expectedToken, token)
			}
		})
	}
}

func testTokenHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "token_request_handler")
}

func testDefaultHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "default_handler")
}

func TestProxy_ReadyZHandler(t *testing.T) {
	tests := []struct {
		name string
		path string
		code int
	}{
		{
			name: "readyz",
			path: "/readyz",
			code: http.StatusOK,
		},
		{
			name: "readyz",
			path: "/readyz/",
			code: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setup()
			defer teardown()

			p := &proxy{logger: mlog.New()}
			rtr.PathPrefix("/readyz").HandlerFunc(p.readyzHandler)

			req, err := http.NewRequest(http.MethodGet, server.URL+test.path, nil)
			if err != nil {
				t.Error(err)
			}

			recorder := httptest.NewRecorder()
			rtr.ServeHTTP(recorder, req)
			if recorder.Code != test.code {
				t.Errorf("Expected code %d, got %d", test.code, recorder.Code)
			}
		})
	}
}

func TestGetScope(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		expected string
	}{
		{
			name:     "resource doesn't have /.default suffix",
			scope:    "https://vault.azure.net",
			expected: "https://vault.azure.net/.default",
		},
		{
			name:     "resource has /.default suffix",
			scope:    "https://vault.azure.net/.default",
			expected: "https://vault.azure.net/.default",
		},
		{
			name:     "resource doesn't  have /.default suffix and has trailing slash",
			scope:    "https://vault.azure.net/",
			expected: "https://vault.azure.net//.default",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope := getScope(test.scope)
			if scope != test.expected {
				t.Errorf("expected scope %s, got %s", test.expected, scope)
			}
		})
	}
}

func TestNewProxy(t *testing.T) {
	testLogger := mlog.New()
	tests := []struct {
		name          string
		tenantID      string
		authorityHost string
		expected      *proxy
		expectedErr   string
	}{
		{
			name:          "tenant id not set",
			tenantID:      "",
			authorityHost: "https://login.microsoftonline.com/",
			expected:      nil,
			expectedErr:   "AZURE_TENANT_ID not set",
		},
		{
			name:          "authority host not set",
			tenantID:      "tenant_id",
			authorityHost: "",
			expected:      nil,
			expectedErr:   "AZURE_AUTHORITY_HOST not set",
		},
		{
			name:          "valid tenant id and authority host",
			tenantID:      "tenant_id",
			authorityHost: "https://login.microsoftonline.com/",
			expected:      &proxy{logger: testLogger, tenantID: "tenant_id", authorityHost: "https://login.microsoftonline.com/", port: 8000},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv(webhook.AzureTenantIDEnvVar, test.tenantID)
			defer os.Unsetenv(webhook.AzureTenantIDEnvVar)

			os.Setenv(webhook.AzureAuthorityHostEnvVar, test.authorityHost)
			defer os.Unsetenv(webhook.AzureAuthorityHostEnvVar)

			got, err := NewProxy(8000, testLogger)
			if err != nil && err.Error() != test.expectedErr {
				t.Errorf("expected error %s, got %s", test.expectedErr, err.Error())
			}
			if err == nil && test.expectedErr != "" {
				t.Errorf("expected error %s, got none", test.expectedErr)
			}
			if test.expected != nil && !reflect.DeepEqual(got, test.expected) {
				t.Errorf("expected proxy %v, got %v", test.expected, got)
			}
		})
	}
}
