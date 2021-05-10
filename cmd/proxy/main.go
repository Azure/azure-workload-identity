package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

const (
	proxyPort        = 8000
	tokenExchangeURL = "https://svctokenexchange.azurewebsites.net/api/token/exchange" // #nosec
)

type proxy struct{}

func main() {
	// TODO add handler to separate default metadata and token request
	if err := http.ListenAndServe(fmt.Sprintf("localhost:%d", proxyPort), &proxy{}); err != nil {
		klog.Fatalf("failed to listen and serve, error: %+v", err)
	}
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// forward the request to the new aad endpoint
	resp, err := p.sendRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	p.writeResponse(w, resp)
}

func (p *proxy) sendRequest(req *http.Request) (*http.Response, error) {
	klog.InfoS("received request", "method", req.Method, "uri", req.RequestURI)
	clientID, resource := parseTokenRequest(req)
	if clientID == "" {
		return &http.Response{Status: strconv.Itoa(http.StatusBadRequest)}, fmt.Errorf("client id is not set")
	}
	if resource == "" {
		resource = "https://management.azure.com/"
	}
	return doTokenRequest(clientID, resource)
}

func (p *proxy) writeResponse(w http.ResponseWriter, res *http.Response) {
	for name, values := range res.Header {
		w.Header()[name] = values
	}
	// Set a special header to notify that the proxy actually serviced the request.
	w.Header().Set("Server", "pi-sidecar-proxy")
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
	res.Body.Close()

	klog.InfoS("request complete", "status", res.StatusCode)
}

func parseTokenRequest(r *http.Request) (clientID, resource string) {
	vals := r.URL.Query()
	if vals != nil {
		clientID = vals.Get("client_id")
		resource = vals.Get("resource")
	}
	return
}

// TODO this will be replaced by the SDK when available
func doTokenRequest(clientID, resource string) (*http.Response, error) {
	// get the service account jwt token
	tokenFile := os.Getenv("TOKEN_FILE_PATH")
	if tokenFile == "" {
		return nil, fmt.Errorf("TOKEN_FILE_PATH not set")
	}
	if _, err := os.Stat(tokenFile); err != nil {
		return nil, fmt.Errorf("token file not found")
	}
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	httpClient := http.Client{}

	reqBody, err := json.Marshal(map[string]string{
		"subjectToken": strings.TrimSuffix(string(token), "\n"),
		"clientId":     clientID,
		"scopes":       resource,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal req body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, tokenExchangeURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	return resp, err
}
