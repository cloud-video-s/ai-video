package adjust

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// EventToken identifies an Adjust event configured in the Adjust dashboard.
type EventToken string

// Business event tokens configured for this application.
const (
	EventTokenPayment      EventToken = "fdhs2r"
	EventTokenOrderCreated EventToken = "6u9gko"
	EventTokenActivation   EventToken = "b159ty"
	EventTokenLogin        EventToken = "qd5uvh"
	EventTokenSubscription EventToken = "9yot8l"
)

// DeviceIDType is the Adjust request parameter used for a device identifier.
type DeviceIDType string

// Device identifier types accepted by the Adjust S2S event endpoint.
const (
	DeviceIDTypeIDFA               DeviceIDType = "idfa"
	DeviceIDTypeGPSADID            DeviceIDType = "gps_adid"
	DeviceIDTypeADID               DeviceIDType = "adid"
	DeviceIDTypeAndroidID          DeviceIDType = "android_id"
	DeviceIDTypeAndroidIDLowerMD5  DeviceIDType = "android_id_lower_md5"
	DeviceIDTypeAndroidIDLowerSHA1 DeviceIDType = "android_id_lower_sha1"
	DeviceIDTypeAndroidIDUpperMD5  DeviceIDType = "android_id_upper_md5"
	DeviceIDTypeAndroidIDUpperSHA1 DeviceIDType = "android_id_upper_sha1"
	DeviceIDTypeIDFV               DeviceIDType = "idfv"
	DeviceIDTypeIMEI               DeviceIDType = "imei"
	DeviceIDTypeIMEILowerMD5       DeviceIDType = "imei_lower_md5"
	DeviceIDTypeMEID               DeviceIDType = "meid"
	DeviceIDTypeWindowsNetworkID   DeviceIDType = "win_naid"
	DeviceIDTypeWindowsHardwareID  DeviceIDType = "win_hwid"
)

var supportedDeviceIDTypes = map[DeviceIDType]struct{}{
	DeviceIDTypeIDFA: {}, DeviceIDTypeGPSADID: {}, DeviceIDTypeADID: {},
	DeviceIDTypeAndroidID: {}, DeviceIDTypeAndroidIDLowerMD5: {},
	DeviceIDTypeAndroidIDLowerSHA1: {}, DeviceIDTypeAndroidIDUpperMD5: {},
	DeviceIDTypeAndroidIDUpperSHA1: {}, DeviceIDTypeIDFV: {}, DeviceIDTypeIMEI: {},
	DeviceIDTypeIMEILowerMD5: {}, DeviceIDTypeMEID: {}, DeviceIDTypeWindowsNetworkID: {},
	DeviceIDTypeWindowsHardwareID: {},
}

// Environment selects the Adjust environment. An empty value uses Adjust's
// default production environment.
type Environment string

const (
	EnvironmentProduction Environment = "production"
	EnvironmentSandbox    Environment = "sandbox"
)

// Event contains the device and optional event data sent to Adjust.
type Event struct {
	IDFA string
	// DeviceIDs must contain at least one supported identifier.
	DeviceIDs map[DeviceIDType]string

	IPAddress string
	UserAgent string
	// CreatedAt is encoded as Adjust's recommended created_at_unix parameter.
	// A zero value lets Adjust use the time at which it receives the event.
	CreatedAt time.Time

	// CallbackParams are included in Adjust raw-data callbacks.
	CallbackParams map[string]string
	// PartnerParams are shared with configured network partners.
	PartnerParams map[string]string

	// Revenue is expressed in full currency units. Currency is required when
	// Revenue is set. A pointer distinguishes an omitted amount from zero.
	Revenue     *float64
	Currency    string
	Environment Environment
}

func (event Event) form(token EventToken, appToken string) (url.Values, error) {
	eventToken := strings.TrimSpace(string(token))
	if eventToken == "" {
		return nil, errors.New("adjust event_token is required")
	}
	if !isAlphaNumeric(eventToken) {
		return nil, errors.New("adjust event_token must be alphanumeric")
	}
	if len(event.DeviceIDs) == 0 {
		return nil, errors.New("adjust event requires at least one device identifier")
	}

	form := url.Values{
		"s2s":         {"1"},
		"app_token":   {appToken},
		"event_token": {eventToken},
	}
	validDeviceIDs := 0
	for identifierType, value := range event.DeviceIDs {
		if _, ok := supportedDeviceIDTypes[identifierType]; !ok {
			return nil, fmt.Errorf("unsupported Adjust device identifier type %q", identifierType)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("Adjust device identifier %q is empty", identifierType)
		}
		form.Set(string(identifierType), value)
		validDeviceIDs++
	}
	if validDeviceIDs == 0 {
		return nil, errors.New("adjust event requires at least one device identifier")
	}

	if ipAddress := strings.TrimSpace(event.IPAddress); ipAddress != "" {
		parsed := net.ParseIP(ipAddress)
		if parsed == nil || parsed.To4() == nil {
			return nil, errors.New("adjust ip_address must be a valid IPv4 address")
		}
		form.Set("ip_address", ipAddress)
	}
	if userAgent := strings.TrimSpace(event.UserAgent); userAgent != "" {
		form.Set("user_agent", userAgent)
	}
	if !event.CreatedAt.IsZero() {
		form.Set("created_at", event.CreatedAt.Format(time.RFC3339))
	}
	if event.Revenue != nil {
		amount := *event.Revenue
		if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0.001 {
			return nil, errors.New("adjust revenue must be a finite value of at least 0.001")
		}
		currency := strings.ToUpper(strings.TrimSpace(event.Currency))
		if !isCurrencyCode(currency) {
			return nil, errors.New("adjust currency must be a three-letter code when revenue is set")
		}
		form.Set("revenue", strconv.FormatFloat(amount, 'f', -1, 64))
		form.Set("currency", currency)
	} else if strings.TrimSpace(event.Currency) != "" {
		return nil, errors.New("adjust revenue is required when currency is set")
	}

	if event.Environment != "" {
		if event.Environment != EnvironmentProduction && event.Environment != EnvironmentSandbox {
			return nil, fmt.Errorf("unsupported Adjust environment %q", event.Environment)
		}
		form.Set("environment", string(event.Environment))
	}

	return form, nil
}

func setJSONParams(form url.Values, name string, parameters map[string]string) error {
	if len(parameters) == 0 {
		return nil
	}
	for key := range parameters {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("adjust %s contains an empty key", name)
		}
	}
	body, err := json.Marshal(parameters)
	if err != nil {
		return fmt.Errorf("encode Adjust %s: %w", name, err)
	}
	form.Set(name, string(body))
	return nil
}

func isCurrencyCode(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return true
}
