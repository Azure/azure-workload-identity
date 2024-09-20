package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/version"
	"github.com/Azure/azure-workload-identity/pkg/webhook"
)

const (
	// "/metadata" portion is case-insensitive in IMDS
	tokenPathPrefix = "/{type:(?i:metadata)}/identity/oauth2/token" // #nosec

	// readyzPathPrefix is the path for readiness probe
	readyzPathPrefix = "/readyz"

	// metadataIPAddress is the IP address of the metadata service
	metadataIPAddress = "169.254.169.254"
	// metadataPort is the port of the metadata service
	metadataPort = 80
	// localhost is the hostname of the localhost
	localhost = "localhost"
)

var (
	userAgent = version.GetUserAgent("proxy")
)

type Proxy interface {
	Run(ctx context.Context) error
}

type proxy struct {
	port          int
	tenantID      string
	authorityHost string
	logger        mlog.Logger
}

// using this from https://github.com/Azure/go-autorest/blob/b3899c1057425994796c92293e931f334af63b4e/autorest/adal/token.go#L1055-L1067
// this struct works with the adal sdks used in clients and azure-cli token requests
type token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	// AAD returns expires_in as a string, ADFS returns it as an int
	ExpiresIn json.Number `json:"expires_in"`
	// expires_on can be in two formats, a UTC time stamp or the number of seconds.
	ExpiresOn string      `json:"expires_on"`
	NotBefore json.Number `json:"not_before"`

	Resource string `json:"resource"`
	Type     string `json:"token_type"`
}

// NewProxy returns a proxy instance
func NewProxy(port int, logger mlog.Logger) (Proxy, error) {
	// tenantID is required for fetching a token using client assertions
	// the mutating webhook will inject the tenantID for the cluster
	tenantID := os.Getenv(webhook.AzureTenantIDEnvVar)
	// authorityHost is required for fetching a token using client assertions
	authorityHost := os.Getenv(webhook.AzureAuthorityHostEnvVar)
	if tenantID == "" {
		return nil, errors.Errorf("%s not set", webhook.AzureTenantIDEnvVar)
	}
	if authorityHost == "" {
		return nil, errors.Errorf("%s not set", webhook.AzureAuthorityHostEnvVar)
	}
	return &proxy{
		port:          port,
		tenantID:      tenantID,
		authorityHost: authorityHost,
		logger:        logger,
	}, nil
}

// Run runs the proxy server
func (p *proxy) Run(ctx context.Context) error {
	rtr := mux.NewRouter()
	rtr.PathPrefix(tokenPathPrefix).HandlerFunc(p.msiHandler)
	rtr.PathPrefix(readyzPathPrefix).HandlerFunc(p.readyzHandler)
	rtr.PathPrefix("/").HandlerFunc(p.defaultPathHandler)

	p.logger.Info("starting the proxy server", "port", p.port, "userAgent", userAgent)
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", localhost, p.port),
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           rtr,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-ctx.Done()

	p.logger.Info("shutting down the proxy server")
	// shutdown the server gracefully with a 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func (p *proxy) msiHandler(w http.ResponseWriter, r *http.Request) {
	p.logger.Info("received token request", "method", r.Method, "uri", r.RequestURI)
	w.Header().Set("Server", userAgent)
	clientID, resource := parseTokenRequest(r)
	// if clientID not found in request, then we default to the AZURE_CLIENT_ID if present.
	// This is to keep consistent with the current behavior in pod identity v1 where we
	// default the client id to the one in AzureIdentity.
	if clientID == "" {
		p.logger.Info("client_id not found in request, defaulting to AZURE_CLIENT_ID", "method", r.Method, "uri", r.RequestURI)
		clientID = os.Getenv(webhook.AzureClientIDEnvVar)
	}

	if clientID == "" {
		http.Error(w, "The client_id parameter or AZURE_CLIENT_ID environment variable must be set", http.StatusBadRequest)
		return
	}
	if resource == "" {
		http.Error(w, "The resource parameter is required.", http.StatusBadRequest)
		return
	}

	// get the token using the msal
	token, err := doTokenRequest(r.Context(), clientID, resource, p.tenantID, p.authorityHost)
	if err != nil {
		p.logger.Error("failed to get token", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.logger.Info("successfully acquired token", "method", r.Method, "uri", r.RequestURI)
	// write the token to the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(token); err != nil {
		p.logger.Error("failed to encode token", err)
	}
}

func (p *proxy) defaultPathHandler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil || req == nil {
		p.logger.Error("failed to create new request", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	host := fmt.Sprintf("%s:%d", metadataIPAddress, metadataPort)
	req.Host = host
	req.URL.Host = host
	req.URL.Scheme = "http"
	if r.Header != nil {
		copyHeader(req.Header, r.Header)
	}
	resp, err := client.Do(req)
	if err != nil {
		p.logger.Error("failed executing request", err, "url", req.URL.String())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Error("failed to read response body", err, "url", req.URL.String())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	p.logger.Info("received response from IMDS", "method", r.Method, "uri", r.RequestURI, "status", resp.StatusCode)

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (p *proxy) readyzHandler(w http.ResponseWriter, r *http.Request) {
	p.logger.Info("received readyz request", "method", r.Method, "uri", r.RequestURI)
	fmt.Fprintf(w, "ok")
}

func doTokenRequest(ctx context.Context, clientID, resource, tenantID, authorityHost string) (*token, error) {
	tokenFilePath := os.Getenv(webhook.AzureFederatedTokenFileEnvVar)
	cred := confidential.NewCredFromAssertionCallback(func(context.Context, confidential.AssertionRequestOptions) (string, error) {
		return readJWTFromFS(tokenFilePath)
	})
	authority, err := url.JoinPath(authorityHost, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct authority URL")
	}

	confidentialClientApp, err := confidential.New(authority, clientID, cred)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create confidential client app")
	}

	result, err := confidentialClientApp.AcquireTokenByCredential(ctx, []string{getScope(resource)})
	if err != nil {
		return nil, errors.Wrap(err, "failed to acquire token")
	}
	return &token{
		AccessToken: result.AccessToken,
		Resource:    resource,
		Type:        "Bearer",
		// -10s is to account for current time changes between the calls
		ExpiresIn: json.Number(strconv.FormatInt(int64(time.Until(result.ExpiresOn)/time.Second)-10, 10)),
		// There is a difference in parsing between the azure sdks and how azure-cli works
		// Using the unix time to be consistent with response from IMDS which works with
		// all the clients.
		ExpiresOn: strconv.FormatInt(result.ExpiresOn.UTC().Unix(), 10),
	}, nil
}

func parseTokenRequest(r *http.Request) (string, string) {
	var clientID, resource string
	if r.URL != nil {
		// Query always return a non-nil map
		clientID = r.URL.Query().Get("client_id")
		resource = r.URL.Query().Get("resource")
	}
	return clientID, resource
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func readJWTFromFS(tokenFilePath string) (string, error) {
	token, err := os.ReadFile(tokenFilePath)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

// ref: https://github.com/AzureAD/microsoft-authentication-library-for-dotnet/issues/747
// For MSAL (v2.0 endpoint) asking an access token for a resource that accepts a v1.0 access token,
// Azure AD parses the desired audience from the requested scope by taking everything before the
// last slash and using it as the resource identifier.
// For example, if the scope is "https://vault.azure.net/.default", the resource identifier is "https://vault.azure.net".
// If the scope is "http://database.windows.net//.default", the resource identifier is "http://database.windows.net/".
func getScope(resource string) string {
	if !strings.HasSuffix(resource, "/.default") {
		resource = resource + "/.default"
	}
	return resource
}
