package middleware

import (
	"context"
)

// APIRequestMetadata contains the authenticated client information that must
// follow a request into service and repository layers. Keeping it as one value
// prevents Gin's private key/value store from becoming the only source of the
// request metadata.
type APIRequestMetadata struct {
	UserID         uint64
	TokenVersion   int64
	LoginType      uint32
	AppCode        string
	AppPackageCode string
	AppVersion     string
	ChannelCode    string
	DeviceCountry  string
	PhoneModel     string
	SystemType     int
}

type apiRequestMetadataContextKey struct{}

// APIContextKey is retained for individual context values used by older
// services. New code should prefer APIRequestMetadataFromContext.
type APIContextKey string

const CtxDeviceCountryKey APIContextKey = "device_country"

// APIRequestMetadataFromContext returns metadata installed by ApiAuth. It can
// be used after a handler passes c.Request.Context() to a Gin-independent
// service.
func APIRequestMetadataFromContext(ctx context.Context) (APIRequestMetadata, bool) {
	if ctx == nil {
		return APIRequestMetadata{}, false
	}
	metadata, ok := ctx.Value(apiRequestMetadataContextKey{}).(APIRequestMetadata)
	return metadata, ok
}

func withAPIRequestMetadata(ctx context.Context, metadata APIRequestMetadata) context.Context {
	ctx = context.WithValue(ctx, apiRequestMetadataContextKey{}, metadata)
	return context.WithValue(ctx, CtxDeviceCountryKey, metadata.DeviceCountry)
}

type UserRepo interface {
	GetAuthState(ctx context.Context, id uint64) (imei string, tokenVersion int64, err error)
}
