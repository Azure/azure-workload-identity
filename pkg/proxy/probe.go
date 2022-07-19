package proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	// retryCount is the number of times to retry probing the proxy.
	retryCount = 7
	// waitTime is the time to wait between retries.
	waitTime = time.Second
	// clientTimeout is the timeout for the client.
	clientTimeout = time.Second * 5
)

// Probe checks if the proxy is ready to serve requests.
func Probe(port int) error {
	url := fmt.Sprintf("http://%s:%d%s", localhost, port, readyzPathPrefix)
	return probe(url)
}

func probe(url string) error {
	client := &http.Client{
		Timeout: clientTimeout,
	}
	for i := 0; i < retryCount; i++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(waitTime)
	}
	return errors.Errorf("failed to probe proxy")
}
