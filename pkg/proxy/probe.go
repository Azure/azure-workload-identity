package proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Probe checks if the proxy is ready to serve requests.
func Probe(port int) error {
	url := fmt.Sprintf("http://%s:%d%s", localhost, port, readyzPathPrefix)
	return probe(url)
}

func probe(url string) error {
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	for i := 0; i < 7; i++ {
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
		time.Sleep(time.Second)
	}
	return errors.Errorf("failed to probe proxy")
}
