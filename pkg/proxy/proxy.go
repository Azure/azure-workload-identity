package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/version"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

const (
	// "/metadata" portion is case-insensitive in IMDS
	tokenPathPrefix = "/{type:(?i:metadata)}/identity/oauth2/token" // #nosec

	// the format for expires_on in UTC with AM/PM
	expiresOnDateFormatPM = "1/2/2006 15:04:05 PM +00:00"
	// the format for expires_on in UTC without AM/PM
	expiresOnDateFormat = "1/2/2006 15:04:05 +00:00"

	// metadataIPAddress is the IP address of the metadata service
	metadataIPAddress = "169.254.169.254"
	// metadataPort is the port of the metadata service
	metadataPort = 80
)

type Proxy interface {
	Run() error
}

type proxy struct {
	port          int
	tenantID      string
	authorityHost string
	logger        logr.Logger
}

// NewProxy returns a proxy instance
func NewProxy(port int, logger logr.Logger) (Proxy, error) {
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
	if logger == nil {
		return nil, errors.New("logger not set")
	}
	return &proxy{
		port:          port,
		tenantID:      tenantID,
		authorityHost: authorityHost,
		logger:        logger,
	}, nil
}

// Run runs the proxy server
func (p *proxy) Run() error {
	rtr := mux.NewRouter()
	rtr.PathPrefix(tokenPathPrefix).HandlerFunc(p.msiHandler)
	rtr.PathPrefix("/").HandlerFunc(p.defaultPathHandler)

	p.logger.Info("starting the proxy server", "port", p.port)
	return http.ListenAndServe(fmt.Sprintf("localhost:%d", p.port), rtr)
}

func (p *proxy) msiHandler(w http.ResponseWriter, r *http.Request) {
	p.logger.Info("received token request", "method", r.Method, "uri", r.RequestURI)
	w.Header().Set("Server", version.GetUserAgent("proxy"))
	clientID, resource := parseTokenRequest(r)
	// TODO (aramase) should we fallback to the clientID in the annotated service account
	// if clientID not found in request? This is to keep consistent with the current behavior
	// in pod identity v1 where we default the client id to the one in AzureIdentity.
	if clientID == "" {
		http.Error(w, "The client_id parameter is required.", http.StatusBadRequest)
		return
	}
	if resource == "" {
		http.Error(w, "The resource parameter is required.", http.StatusBadRequest)
		return
	}

	// get the token using the msal
	token, err := doTokenRequest(r.Context(), clientID, resource, p.tenantID, p.authorityHost)
	if err != nil {
		p.logger.Error(err, "failed to get token")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.logger.Info("successfully acquired token", "method", r.Method, "uri", r.RequestURI)
	// write the token to the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(token); err != nil {
		p.logger.Error(err, "failed to encode token")
	}
}

func (p *proxy) defaultPathHandler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil || req == nil {
		p.logger.Error(err, "failed to create new request")
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
		p.logger.Error(err, "failed executing request", "url", req.URL.String())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Error(err, "failed to read response body", "url", req.URL.String())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	p.logger.Info("received response from IMDS", "method", r.Method, "uri", r.RequestURI, "status", resp.StatusCode)

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func doTokenRequest(ctx context.Context, clientID, resource, tenantID, authorityHost string) (*adal.Token, error) {
	tokenFilePath := os.Getenv(webhook.AzureFederatedTokenFileEnvVar)
	signedAssertion, err := readJWTFromFS(tokenFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read service account token")
	}

	cred, err := confidential.NewCredFromAssertion(signedAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create confidential creds")
	}

	confidentialClientApp, err := confidential.New(clientID, cred,
		confidential.WithAuthority(fmt.Sprintf("%s%s/oauth2/token", authorityHost, tenantID)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create confidential client app")
	}

	scope := strings.TrimSuffix(resource, "/")
	if !strings.HasPrefix(scope, "/.default") {
		scope = scope + "/.default"
	}
	result, err := confidentialClientApp.AcquireTokenByCredential(ctx, []string{scope})
	if err != nil {
		return nil, errors.Wrap(err, "failed to acquire token")
	}

	token := &adal.Token{}
	token.AccessToken = result.AccessToken
	token.Resource = resource
	token.Type = "Bearer"
	token.ExpiresOn, err = parseExpiresOn(result.ExpiresOn.UTC().Local().Format(expiresOnDateFormat))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse expires_on")
	}
	return token, nil
}

// Vendored from https://github.com/Azure/go-autorest/blob/def88ef859fb980eff240c755a70597bc9b490d0/autorest/adal/token.go
// converts expires_on to the number of seconds
func parseExpiresOn(s string) (json.Number, error) {
	// convert the expiration date to the number of seconds from now
	timeToDuration := func(t time.Time) json.Number {
		dur := t.Sub(time.Now().UTC())
		return json.Number(strconv.FormatInt(int64(dur.Round(time.Second).Seconds()), 10))
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		// this is the number of seconds case, no conversion required
		return json.Number(s), nil
	} else if eo, err := time.Parse(expiresOnDateFormatPM, s); err == nil {
		return timeToDuration(eo), nil
	} else if eo, err := time.Parse(expiresOnDateFormat, s); err == nil {
		return timeToDuration(eo), nil
	} else {
		// unknown format
		return json.Number(""), err
	}
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
