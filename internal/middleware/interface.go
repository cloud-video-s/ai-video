package middleware

import (
	"context"
)

// CtxDeviceCountryKey is used by request middleware to pass the resolved
// device country into API services without coupling them to Gin.
const CtxDeviceCountryKey = "device_country"

type UserRepo interface {
	GetAuthState(ctx context.Context, id uint64) (imei string, tokenVersion int64, err error)
}
