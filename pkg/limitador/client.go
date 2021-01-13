package limitador

import (
	"bytes"
	"fmt"
	limitadorv1alpha1 "github.com/3scale/limitador-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/url"
)

type Client struct {
	url url.URL
}

func NewClient(url url.URL) Client {
	return Client{url: url}
}

func (client *Client) CreateLimit(rateLimitSpec *limitadorv1alpha1.RateLimitSpec) error {
	jsonLimit, err := json.Marshal(rateLimitSpec)
	if err != nil {
		return err
	}

	_, err = http.Post(
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

	httpClient := &http.Client{}
	_, err = httpClient.Do(req)

	return err
}
