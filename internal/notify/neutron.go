package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type NeutronClient struct {
	apiURL   string
	baseURL  string
	client   *http.Client
}

func New(apiURL, baseURL string) *NeutronClient {
	return &NeutronClient{
		apiURL:  apiURL,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *NeutronClient) SendReportLink(jobName, buildID string) {
	if n.apiURL == "" {
		return
	}

	reportURL := fmt.Sprintf("%s/run/%s/%s", n.baseURL, jobName, buildID)
	payload := map[string]string{
		"report_url": reportURL,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("notify: marshal payload: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/report/%s/link", n.apiURL, jobName)
	resp, err := n.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("notify: post to neutron %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("notify: neutron returned %d for %s", resp.StatusCode, url)
	} else {
		log.Printf("notify: report link sent to neutron for %s/%s → %s", jobName, buildID, reportURL)
	}
}
