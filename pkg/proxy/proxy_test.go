package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azure/aad-pod-managed-identity/pkg/webhook"

	"github.com/gorilla/mux"
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
			expectedBody:       "The client_id parameter is required.\n",
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

			p := &proxy{}
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

// Vendored from https://github.com/Azure/go-autorest/blob/def88ef859fb980eff240c755a70597bc9b490d0/autorest/adal/token_test.go
func TestParseExpiresOn(t *testing.T) {
	// get current time, round to nearest second, and add one hour
	n := time.Now().UTC().Round(time.Second).Add(time.Hour)
	amPM := "AM"
	if n.Hour() >= 12 {
		amPM = "PM"
	}
	testcases := []struct {
		Name   string
		String string
		Value  int64
	}{
		{
			Name:   "integer",
			String: "3600",
			Value:  3600,
		},
		{
			Name:   "timestamp with AM/PM",
			String: fmt.Sprintf("%d/%d/%d %d:%02d:%02d %s +00:00", n.Month(), n.Day(), n.Year(), n.Hour(), n.Minute(), n.Second(), amPM),
			Value:  3600,
		},
		{
			Name:   "timestamp without AM/PM",
			String: fmt.Sprintf("%d/%d/%d %d:%02d:%02d +00:00", n.Month(), n.Day(), n.Year(), n.Hour(), n.Minute(), n.Second()),
			Value:  3600,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(subT *testing.T) {
			jn, err := parseExpiresOn(tc.String)
			if err != nil {
				subT.Error(err)
			}
			i, err := jn.Int64()
			if err != nil {
				subT.Error(err)
			}
			if i != tc.Value {
				subT.Logf("expected %d, got %d", tc.Value, i)
				subT.Fail()
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
				if err := ioutil.WriteFile(tokenFilePath, []byte("token"), 0600); err != nil {
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

func testTokenHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "token_request_handler")
}

func testDefaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "default_handler")
}
