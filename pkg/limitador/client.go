package limitador

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"k8s.io/apimachinery/pkg/util/json"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/helpers"
	"github.com/kuadrant/limitador-operator/pkg/log"
)

type Client struct {
	httpClient *http.Client
	url        url.URL
}

func NewClient(url url.URL) Client {
	var transport http.RoundTripper
	if log.Log.V(1).Enabled() {
		transport = &helpers.VerboseTransport{}
	}

	return Client{
		url:        url,
		httpClient: &http.Client{Transport: transport},
	}
}

func (client *Client) CreateLimit(rateLimitSpec *limitadorv1alpha1.RateLimitSpec) error {
	jsonLimit, err := json.Marshal(rateLimitSpec)
	if err != nil {
		return err
	}

	_, err = client.httpClient.Post(
		fmt.Sprintf("%s/limits", client.url.String()),
		"application/json",
		bytes.NewBuffer(jsonLimit),
	)

	return err
}

func (client *Client) DeleteLimit(rateLimitSpec *limitadorv1alpha1.RateLimitSpec) error {
	jsonLimit, err := json.Marshal(rateLimitSpec)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/limits", client.url.String()),
		bytes.NewBuffer(jsonLimit),
	)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	_, err = client.httpClient.Do(req)

	return err
}
