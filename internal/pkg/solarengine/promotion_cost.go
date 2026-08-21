package solarengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

const promotionCostEndpoint = "finance/promotion"

// ResponseCode is a SolarEngine application-level response code.
type ResponseCode int

const (
	ResponseCodeSuccess             ResponseCode = 200
	ResponseCodeInvalidSignature    ResponseCode = 300
	ResponseCodeInvalidParameter    ResponseCode = 400
	ResponseCodeMultiProductMetrics ResponseCode = 401
	ResponseCodePermissionDenied    ResponseCode = 403
	ResponseCodeDateRangeExceeded   ResponseCode = 409
	ResponseCodePageSizeExceeded    ResponseCode = 410
	ResponseCodeTooManyRequests     ResponseCode = 5000010
)

// Currency is a currency value accepted by SolarEngine reports.
type Currency string

const (
	CurrencyCNY Currency = "1"
	CurrencyUSD Currency = "2"
	CurrencyNGN Currency = "3"
	CurrencyBRL Currency = "4"
	CurrencyPKR Currency = "5"
	CurrencyAED Currency = "6"
	CurrencyVND Currency = "7"
	CurrencyHKD Currency = "8"
	CurrencyINR Currency = "9"
	CurrencyGBP Currency = "10"
	CurrencyAUD Currency = "11"
	CurrencyJPY Currency = "12"
)

// OSPlatform is an operating system filter accepted by the report API.
type OSPlatform string

const (
	OSPlatformAndroid OSPlatform = "ANDROID"
	OSPlatformIOS     OSPlatform = "IOS"
)

// PromotionCostDimension is a group-by field supported by the promotion cost
// report.
type PromotionCostDimension string

const (
	PromotionCostDimensionDate              PromotionCostDimension = "date"
	PromotionCostDimensionPromotionPlatform PromotionCostDimension = "promotion_platform"
	PromotionCostDimensionAccountID         PromotionCostDimension = "account_id"
	PromotionCostDimensionProductName       PromotionCostDimension = "product_name"
	PromotionCostDimensionAppName           PromotionCostDimension = "app_name"
	PromotionCostDimensionAppID             PromotionCostDimension = "app_id"
	PromotionCostDimensionOSPlatform        PromotionCostDimension = "os_platform"
	PromotionCostDimensionCountryCode       PromotionCostDimension = "country_code"
	PromotionCostDimensionOriginalCurrency  PromotionCostDimension = "original_currency"
)

// PromotionCostMetric is a metric supported by the promotion cost report.
type PromotionCostMetric string

const (
	PromotionCostMetricCost         PromotionCostMetric = "cost"
	PromotionCostMetricOriginalCost PromotionCostMetric = "original_cost"
)

// OrderType controls report sorting.
type OrderType string

const (
	OrderTypeAscending  OrderType = "ASC"
	OrderTypeDescending OrderType = "DESC"
)

// PromotionCostFilter limits the records included in a promotion cost report.
type PromotionCostFilter struct {
	Currency           Currency     `json:"currency,omitempty"`
	PromotionPlatforms []string     `json:"promotion_platforms,omitempty"`
	AccountIDs         []string     `json:"account_ids,omitempty"`
	ProductIDs         []string     `json:"product_ids,omitempty"`
	AppIDs             []string     `json:"app_ids,omitempty"`
	OSPlatforms        []OSPlatform `json:"os_platforms,omitempty"`
	CountryCodes       []string     `json:"country_codes,omitempty"`
}

// PromotionCostRequest is the body of a promotion cost report query. Page and
// PageSize use zero to let SolarEngine apply its documented defaults.
type PromotionCostRequest struct {
	StartDate string                   `json:"start_date"`
	EndDate   string                   `json:"end_date"`
	Filter    *PromotionCostFilter     `json:"filter,omitempty"`
	GroupBy   []PromotionCostDimension `json:"group_by,omitempty"`
	Metrics   []PromotionCostMetric    `json:"metrics,omitempty"`
	OrderBy   PromotionCostMetric      `json:"order_by,omitempty"`
	OrderType OrderType                `json:"order_type,omitempty"`
	Page      int                      `json:"page,omitempty"`
	PageSize  int                      `json:"page_size,omitempty"`
}

// PageInfo contains promotion cost report pagination metadata.
type PageInfo struct {
	TotalNumber int `json:"total_number"`
	Page        int `json:"page"`
	PageSize    int `json:"page_size"`
	TotalPage   int `json:"total_page"`
}

// ReportRow contains the dynamic dimension and metric columns requested by
// the caller. Decode can be used to read a field into a concrete Go type.
type ReportRow map[string]json.RawMessage

// Decode decodes one dynamic report field into destination.
func (r ReportRow) Decode(field string, destination any) error {
	if destination == nil {
		return errors.New("SolarEngine report row destination is required")
	}
	raw, ok := r[field]
	if !ok {
		return fmt.Errorf("SolarEngine report row field %q is missing", field)
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		return fmt.Errorf("decode SolarEngine report row field %q: %w", field, err)
	}
	return nil
}

// PromotionCostData contains page metadata and the requested dynamic rows.
type PromotionCostData struct {
	PageInfo PageInfo    `json:"page_info"`
	List     []ReportRow `json:"list"`
}

// PromotionCostResponse is the response envelope returned by SolarEngine.
type PromotionCostResponse struct {
	Code      ResponseCode      `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id"`
	Count     int               `json:"count"`
	Data      PromotionCostData `json:"data"`
}

// QueryPromotionCostReport returns SolarEngine finance promotion cost data.
//
// SolarEngine endpoint: POST /finance/promotion.
func (c *Client) QueryPromotionCostReport(
	ctx context.Context,
	request PromotionCostRequest,
) (*PromotionCostResponse, error) {
	request, err := normalizePromotionCostRequest(request)
	if err != nil {
		return nil, err
	}
	raw, statusCode, err := c.postReport(
		ctx,
		promotionCostEndpoint,
		request.StartDate,
		request.EndDate,
		request,
	)
	if err != nil {
		return nil, err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, parseAPIError(statusCode, raw)
	}

	var response PromotionCostResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode SolarEngine promotion cost response: %w", err)
	}
	if response.Code != ResponseCodeSuccess {
		return nil, &APIError{
			StatusCode: statusCode,
			Code:       response.Code,
			Message:    response.Message,
			RequestID:  response.RequestID,
		}
	}
	if response.Data.List == nil {
		response.Data.List = []ReportRow{}
	}
	return &response, nil
}

func normalizePromotionCostRequest(request PromotionCostRequest) (PromotionCostRequest, error) {
	request.StartDate = strings.TrimSpace(request.StartDate)
	request.EndDate = strings.TrimSpace(request.EndDate)
	start, err := parseReportDate("start_date", request.StartDate)
	if err != nil {
		return PromotionCostRequest{}, err
	}
	end, err := parseReportDate("end_date", request.EndDate)
	if err != nil {
		return PromotionCostRequest{}, err
	}
	minimumDate := time.Date(2022, time.June, 20, 0, 0, 0, 0, time.UTC)
	if start.Before(minimumDate) || end.Before(minimumDate) {
		return PromotionCostRequest{}, errors.New("SolarEngine report dates must not be before 2022-06-20")
	}
	if end.Before(start) {
		return PromotionCostRequest{}, errors.New("SolarEngine end_date must not be before start_date")
	}
	if request.Page < 0 {
		return PromotionCostRequest{}, errors.New("SolarEngine page must be a positive integer")
	}
	if request.PageSize < 0 || request.PageSize > 1000 {
		return PromotionCostRequest{}, errors.New("SolarEngine page_size must be between 1 and 1000")
	}

	if request.Filter != nil {
		filter := *request.Filter
		filter.Currency = Currency(strings.TrimSpace(string(filter.Currency)))
		if filter.Currency != "" && !slices.Contains([]Currency{
			CurrencyCNY, CurrencyUSD, CurrencyNGN, CurrencyBRL, CurrencyPKR, CurrencyAED,
			CurrencyVND, CurrencyHKD, CurrencyINR, CurrencyGBP, CurrencyAUD, CurrencyJPY,
		}, filter.Currency) {
			return PromotionCostRequest{}, fmt.Errorf("unsupported SolarEngine currency %q", filter.Currency)
		}
		filter.PromotionPlatforms, err = normalizeStringList("promotion_platforms", filter.PromotionPlatforms)
		if err != nil {
			return PromotionCostRequest{}, err
		}
		filter.AccountIDs, err = normalizeStringList("account_ids", filter.AccountIDs)
		if err != nil {
			return PromotionCostRequest{}, err
		}
		filter.ProductIDs, err = normalizeStringList("product_ids", filter.ProductIDs)
		if err != nil {
			return PromotionCostRequest{}, err
		}
		filter.AppIDs, err = normalizeStringList("app_ids", filter.AppIDs)
		if err != nil {
			return PromotionCostRequest{}, err
		}
		filter.CountryCodes, err = normalizeStringList("country_codes", filter.CountryCodes)
		if err != nil {
			return PromotionCostRequest{}, err
		}
		filter.OSPlatforms = slices.Clone(filter.OSPlatforms)
		for index, platform := range filter.OSPlatforms {
			platform = OSPlatform(strings.ToUpper(strings.TrimSpace(string(platform))))
			if platform != OSPlatformAndroid && platform != OSPlatformIOS {
				return PromotionCostRequest{}, fmt.Errorf("unsupported SolarEngine os_platforms[%d] %q", index, platform)
			}
			filter.OSPlatforms[index] = platform
		}
		request.Filter = &filter
	}

	request.GroupBy = slices.Clone(request.GroupBy)
	for index, dimension := range request.GroupBy {
		dimension = PromotionCostDimension(strings.TrimSpace(string(dimension)))
		if !isPromotionCostDimension(dimension) {
			return PromotionCostRequest{}, fmt.Errorf("unsupported SolarEngine group_by[%d] %q", index, dimension)
		}
		request.GroupBy[index] = dimension
	}
	request.Metrics = slices.Clone(request.Metrics)
	for index, metric := range request.Metrics {
		metric = PromotionCostMetric(strings.TrimSpace(string(metric)))
		if !isPromotionCostMetric(metric) {
			return PromotionCostRequest{}, fmt.Errorf("unsupported SolarEngine metrics[%d] %q", index, metric)
		}
		request.Metrics[index] = metric
	}
	request.OrderBy = PromotionCostMetric(strings.TrimSpace(string(request.OrderBy)))
	if request.OrderBy != "" && !isPromotionCostMetric(request.OrderBy) {
		return PromotionCostRequest{}, fmt.Errorf("unsupported SolarEngine order_by %q", request.OrderBy)
	}
	request.OrderType = OrderType(strings.ToUpper(strings.TrimSpace(string(request.OrderType))))
	if request.OrderType != "" && request.OrderType != OrderTypeAscending && request.OrderType != OrderTypeDescending {
		return PromotionCostRequest{}, fmt.Errorf("unsupported SolarEngine order_type %q", request.OrderType)
	}
	return request, nil
}

func parseReportDate(name, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("SolarEngine %s is required", name)
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("SolarEngine %s must use YYYY-MM-DD format", name)
	}
	return parsed, nil
}

func normalizeStringList(name string, values []string) ([]string, error) {
	values = slices.Clone(values)
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("SolarEngine %s[%d] must not be empty", name, index)
		}
		values[index] = value
	}
	return values, nil
}

func isPromotionCostDimension(value PromotionCostDimension) bool {
	return slices.Contains([]PromotionCostDimension{
		PromotionCostDimensionDate,
		PromotionCostDimensionPromotionPlatform,
		PromotionCostDimensionAccountID,
		PromotionCostDimensionProductName,
		PromotionCostDimensionAppName,
		PromotionCostDimensionAppID,
		PromotionCostDimensionOSPlatform,
		PromotionCostDimensionCountryCode,
		PromotionCostDimensionOriginalCurrency,
	}, value)
}

func isPromotionCostMetric(value PromotionCostMetric) bool {
	return value == PromotionCostMetricCost || value == PromotionCostMetricOriginalCost
}
