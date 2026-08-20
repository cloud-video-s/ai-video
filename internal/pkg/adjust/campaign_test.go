package adjust

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetTrackers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", request.Method)
		}
		if request.URL.Path != "/apps/yxs12pfewq/trackers" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.URL.Query().Get("cursor"); got != "next cursor" {
			t.Errorf("cursor = %q", got)
		}
		if got := request.URL.Query().Get("limit"); got != "25" {
			t.Errorf("limit = %q", got)
		}
		if got := request.Header.Get("Authorization"); got != "Token token=test-api-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"paging": {"cursor": "cursor-2", "next": "https://api.adjust.test/next"},
				"items": [{
					"name": "Adroll", "token": "abc123", "label": "Adroll", "level": 1,
					"archived": false, "has_subtrackers": true, "partner_id": 3,
					"cost_data_enabled": false, "url": "https://app.adjust.com/abc123",
					"click_url": "https://app.adjust.com/abc123?campaign={campaign_name}",
					"impression_url": "https://s2s.adjust.com/impression/abc123"
				}]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{APIToken: "test-api-token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.GetTrackers(context.Background(), " yxs12pfewq ", ListOptions{
		Cursor: " next cursor ",
		Limit:  25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Data.Paging.Cursor != "cursor-2" || len(response.Data.Items) != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
	tracker := response.Data.Items[0]
	if tracker.Token != "abc123" || tracker.Level != 1 || !tracker.HasSubtrackers ||
		tracker.PartnerID == nil || *tracker.PartnerID != 3 {
		t.Fatalf("unexpected tracker: %#v", tracker)
	}
}

func TestGetTrackerChildren(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/apps/gwzpeepw8uf8/trackers/abc123/children" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if request.URL.RawQuery != "" {
			t.Errorf("query = %q, want empty", request.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"paging": {"cursor": "", "next": ""},
				"items": [{
					"name": "Adroll::SpringCampaign", "token": "xyz456", "label": "SpringCampaign",
					"level": 2, "archived": false, "has_subtrackers": false, "partner_id": null,
					"cost_data_enabled": false, "url": "https://app.adjust.com/xyz456",
					"click_url": "https://app.adjust.com/xyz456", "impression_url": "https://s2s.adjust.com/impression/xyz456"
				}]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{APIToken: "test-api-token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.GetTrackerChildren(
		context.Background(), "gwzpeepw8uf8", "abc123", ListOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Items) != 1 || response.Data.Items[0].PartnerID != nil ||
		response.Data.Items[0].Level != 2 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestGetTrackersReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"error": {"code": "invalid_token", "message": "API token is invalid"},
			"request_id": "request-1"
		}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{APIToken: "bad-token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetTrackers(context.Background(), "gwzpeepw8uf8", ListOptions{})
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("error = %#v, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusUnauthorized || apiError.Code != "invalid_token" ||
		apiError.Message != "API token is invalid" || apiError.RequestID != "request-1" {
		t.Fatalf("unexpected API error: %#v", apiError)
	}
}

func TestGetTrackersValidatesInput(t *testing.T) {
	client, err := NewClient(ClientConfig{APIToken: "test-api-token"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		appToken string
		options  ListOptions
		want     string
	}{
		{name: "missing app token", want: "app_token is required"},
		{name: "invalid app token", appToken: "bad/token", want: "app_token must be alphanumeric"},
		{name: "negative limit", appToken: "gwzpeepw8uf8", options: ListOptions{Limit: -1}, want: "limit must be a positive integer"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := client.GetTrackers(context.Background(), test.appToken, test.options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}

	_, err = client.GetTrackerChildren(context.Background(), "gwzpeepw8uf8", "bad/token", ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "link_token must be alphanumeric") {
		t.Fatalf("error = %v", err)
	}
}

func TestNewClientValidatesConfig(t *testing.T) {
	tests := []struct {
		name   string
		config ClientConfig
		want   string
	}{
		{name: "missing API token", config: ClientConfig{}, want: "API token is required"},
		{name: "invalid base URL", config: ClientConfig{APIToken: "token", BaseURL: "://bad"}, want: "base URL is invalid"},
		{name: "base URL credentials", config: ClientConfig{APIToken: "token", BaseURL: "https://user@example.com"}, want: "must not contain credentials"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewClient(test.config)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}
