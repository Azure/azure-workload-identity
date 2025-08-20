package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// clientAssertionCredential authenticates an application with assertions provided by a callback function.
type clientAssertionCredential struct {
	assertion     string
	tokenFile     string
	tokenEndpoint string
	lastRead      time.Time
	clientID      string
	tokenClient   *http.Client
}

// clientAssertionCredentialOptions contains optional parameters for ClientAssertionCredential.
type clientAssertionCredentialOptions struct {
	azcore.ClientOptions
}

func createTokenHTTPClient(sni string, caFile string) (*http.Client, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file %q: %w", caFile, err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certs from PEM from %q", caFile)
	}

	tlsConfig := &tls.Config{
		ServerName: sni,
		RootCAs:    caCertPool,
	}

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is not of type *http.Transport")
	}
	transportWithTLSConfigOverride := defaultTransport.Clone()
	transportWithTLSConfigOverride.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transportWithTLSConfigOverride,
	}, nil
}

// newClientAssertionCredential constructs a clientAssertionCredential. Pass nil for options to accept defaults.
func newClientAssertionCredential(
	clientID string, tokenEndpoint string, sni string,
	caFile string, tokenFile string,
	options *clientAssertionCredentialOptions,
) (*clientAssertionCredential, error) {
	tokenHTTPClient, err := createTokenHTTPClient(sni, caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create token transport: %w", err)
	}

	c := &clientAssertionCredential{
		clientID:      clientID,
		tokenFile:     tokenFile,
		tokenEndpoint: tokenEndpoint,
		tokenClient:   tokenHTTPClient,
	}

	if options == nil {
		options = &clientAssertionCredentialOptions{}
	}

	return c, nil
}

// GetToken implements the TokenCredential interface
func (c *clientAssertionCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	assertion, err := c.getAssertion(ctx)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("failed to get assertion: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.tokenEndpoint, nil)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("failed to create request: %w", err)
	}
	q := url.Values{}
	q.Add("grant_type", "client_credentials")
	q.Add("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	q.Add("scope", strings.Join(opts.Scopes, " "))
	q.Add("client_assertion", assertion)
	q.Add("client_id", c.clientID)
	req.URL.RawQuery = q.Encode()

	resp, err := c.tokenClient.Do(req)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return azcore.AccessToken{}, fmt.Errorf("unexpected status code %d from token endpoint", resp.StatusCode)
	}

	// see oauth/ops.TokenResponse
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return azcore.AccessToken{}, fmt.Errorf("failed to decode token response: %w", err)
	}

	return azcore.AccessToken{
		Token:     tokenResponse.AccessToken,
		ExpiresOn: time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
	}, nil
}

// getAssertion reads the assertion from the file and returns it
// if the file has not been read in the last 5 minutes
func (c *clientAssertionCredential) getAssertion(context.Context) (string, error) {
	if now := time.Now(); c.lastRead.Add(5 * time.Minute).Before(now) {
		content, err := os.ReadFile(c.tokenFile)
		if err != nil {
			return "", err
		}
		c.assertion = string(content)
		c.lastRead = now
	}
	return c.assertion, nil
}
