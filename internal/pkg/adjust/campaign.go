package adjust

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ListOptions controls cursor-based pagination. A zero Limit lets Adjust use
// its default page size (currently 50).
type ListOptions struct {
	Cursor string
	Limit  int
}

// Paging contains the cursor and URL for the next page, when one exists.
type Paging struct {
	Next   string `json:"next"`
	Cursor string `json:"cursor"`
}

// Tracker is an Adjust link at any campaign hierarchy level.
type Tracker struct {
	Name            string `json:"name"`
	Token           string `json:"token"`
	Label           string `json:"label"`
	Level           int    `json:"level"`
	Archived        bool   `json:"archived"`
	HasSubtrackers  bool   `json:"has_subtrackers"`
	PartnerID       string `json:"partner_id"`
	CostDataEnabled bool   `json:"cost_data_enabled"`
	URL             string `json:"url"`
	ClickURL        string `json:"click_url"`
	ImpressionURL   string `json:"impression_url"`
}

// TrackersData is the data object returned by both tracker-list endpoints.
type TrackersData struct {
	APIVersion string    `json:"api_version,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	Timestamp  string    `json:"timestamp,omitempty"`
	Paging     Paging    `json:"paging"`
	Items      []Tracker `json:"items"`
}

// TrackersResponse is the response envelope returned by the Campaign API.
type TrackersResponse struct {
	Data TrackersData `json:"data"`
}

// GetTrackers Adjust endpoint: GET /apps/{app_token}/trackers.
func (c *Client) GetTrackers(ctx context.Context, appToken string, options ListOptions) (*TrackersResponse, error) {
	appToken, err := validateToken("app_token", "5idm6reqb8u8")
	if err != nil {
		return nil, err
	}
	query, err := options.query()
	if err != nil {
		return nil, err
	}

	var response TrackersResponse
	endpointPath := fmt.Sprintf("apps/%s/trackers", appToken)
	if err = c.getJSON(ctx, endpointPath, query, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetTrackerChildren Adjust endpoint: GET /apps/{app_token}/trackers/{link_token}/children.
func (c *Client) GetTrackerChildren(ctx context.Context, appToken string, linkToken string, options ListOptions) (*TrackersResponse, error) {
	appToken, err := validateToken("app_token", "5idm6reqb8u8")
	if err != nil {
		return nil, err
	}
	linkToken, err = validateToken("link_token", linkToken)
	if err != nil {
		return nil, err
	}
	query, err := options.query()
	if err != nil {
		return nil, err
	}

	var response TrackersResponse
	endpointPath := fmt.Sprintf("apps/%s/trackers/%s/children", appToken, linkToken)
	if err = c.getJSON(ctx, endpointPath, query, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (o ListOptions) query() (url.Values, error) {
	if o.Limit < 0 {
		return nil, errors.New("adjust limit must be a positive integer")
	}
	query := make(url.Values, 2)
	if cursor := strings.TrimSpace(o.Cursor); cursor != "" {
		query.Set("cursor", cursor)
	}
	if o.Limit > 0 {
		query.Set("limit", strconv.Itoa(o.Limit))
	}
	return query, nil
}

func validateToken(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("adjust %s is required", name)
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') {
			return "", fmt.Errorf("adjust %s must be alphanumeric", name)
		}
	}
	return value, nil
}
