package limitador

import (
	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCreateLimit(t *testing.T) {
	rateLimitSpec := exampleRateLimitSpec()
	rateLimitSpecJson, err := json.Marshal(rateLimitSpec)
	assert.NoError(t, err)

	testServerUrl, closeServerFunc := newTestServer(t, "POST", "/limits", string(rateLimitSpecJson))
	defer closeServerFunc()

	limitadorClient := NewClient(*testServerUrl)
	err = limitadorClient.CreateLimit(rateLimitSpec)

	assert.NoError(t, err)
}

func TestDeleteLimit(t *testing.T) {
	rateLimitSpec := exampleRateLimitSpec()
	rateLimitSpecJson, err := json.Marshal(rateLimitSpec)
	assert.NoError(t, err)

	testServerUrl, closeServerFunc := newTestServer(t, "DELETE", "/limits", string(rateLimitSpecJson))
	defer closeServerFunc()

	limitadorClient := NewClient(*testServerUrl)
	err = limitadorClient.DeleteLimit(rateLimitSpec)

	assert.NoError(t, err)
}

// Creates a test server that checks the given HTTP request fields
func newTestServer(t *testing.T, expectedMethod string, expectedPath string, expectedBody string) (*url.URL, func()) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedMethod, r.Method)

		assert.Equal(t, expectedPath, r.URL.Path)

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		err = r.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(body))
	}))

	serverUrl, err := url.Parse(testServer.URL)
	assert.Nil(t, err)

	return serverUrl, testServer.Close
}

func exampleRateLimitSpec() *limitadorv1alpha1.RateLimitSpec {
	return &limitadorv1alpha1.RateLimitSpec{
		Conditions: []string{"req.method == GET"},
		MaxValue:   10,
		Namespace:  "test-namespace",
		Seconds:    60,
		Variables:  []string{"user_id"},
	}
}
