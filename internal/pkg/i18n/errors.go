package i18n

import (
	"net/http"
	"strings"

	"ai-video/internal/pkg/errcode"

	"github.com/gin-gonic/gin"
)

const (
	LocaleZhCN = "zh-CN"
	LocaleEnUS = "en-US"
	LocaleJaJP = "ja-JP"
	LocaleKoKR = "ko-KR"
	LocaleEsES = "es-ES"
)

const (
	contextLocaleKey = "api_locale"
	contextAPIKey    = "api_localized_errors"
)

var messages = map[string]map[string]string{
	LocaleZhCN: {
		"invalid_request": "请求参数有误，请检查后重试", "unauthorized": "登录状态已失效，请重新登录",
		"forbidden": "暂无权限执行此操作", "not_found": "请求的内容不存在",
		"server_error": "服务繁忙，请稍后重试", "file_too_large": "文件大小超过允许上限",
		"unsupported_file": "不支持该文件类型", "conflict": "当前操作存在冲突，请刷新后重试",
	},
	LocaleEnUS: {
		"invalid_request": "Invalid request. Please check your input and try again.", "unauthorized": "Your session has expired. Please sign in again.",
		"forbidden": "You do not have permission to perform this action.", "not_found": "The requested resource was not found.",
		"server_error": "The service is temporarily unavailable. Please try again later.", "file_too_large": "The file exceeds the allowed size limit.",
		"unsupported_file": "This file type is not supported.", "conflict": "The operation conflicts with the current state. Please refresh and try again.",
	},
	LocaleJaJP: {
		"invalid_request": "リクエスト内容を確認して、もう一度お試しください。", "unauthorized": "ログインの有効期限が切れました。再度ログインしてください。",
		"forbidden": "この操作を実行する権限がありません。", "not_found": "要求されたデータが見つかりません。",
		"server_error": "サービスが混み合っています。しばらくしてからお試しください。", "file_too_large": "ファイルサイズが上限を超えています。",
		"unsupported_file": "このファイル形式はサポートされていません。", "conflict": "現在の状態と競合しています。更新してから再度お試しください。",
	},
	LocaleKoKR: {
		"invalid_request": "요청 정보를 확인한 후 다시 시도해 주세요.", "unauthorized": "로그인이 만료되었습니다. 다시 로그인해 주세요.",
		"forbidden": "이 작업을 수행할 권한이 없습니다.", "not_found": "요청한 항목을 찾을 수 없습니다.",
		"server_error": "서비스가 일시적으로 원활하지 않습니다. 잠시 후 다시 시도해 주세요.", "file_too_large": "파일 크기가 허용 한도를 초과했습니다.",
		"unsupported_file": "지원하지 않는 파일 형식입니다.", "conflict": "현재 상태와 충돌합니다. 새로고침 후 다시 시도해 주세요.",
	},
	LocaleEsES: {
		"invalid_request": "La solicitud no es válida. Revisa los datos e inténtalo de nuevo.", "unauthorized": "La sesión ha caducado. Inicia sesión de nuevo.",
		"forbidden": "No tienes permiso para realizar esta acción.", "not_found": "No se encontró el recurso solicitado.",
		"server_error": "El servicio no está disponible temporalmente. Inténtalo más tarde.", "file_too_large": "El archivo supera el tamaño permitido.",
		"unsupported_file": "Este tipo de archivo no es compatible.", "conflict": "La operación entra en conflicto con el estado actual. Actualiza e inténtalo de nuevo.",
	},
}

func SupportedLocales() []string {
	return []string{LocaleZhCN, LocaleEnUS, LocaleJaJP, LocaleKoKR, LocaleEsES}
}

func NormalizeLocale(value string) string {
	value = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "_", "-")))
	if index := strings.Index(value, ","); index >= 0 {
		value = value[:index]
	}
	if index := strings.Index(value, ";"); index >= 0 {
		value = value[:index]
	}
	switch {
	case strings.HasPrefix(value, "en"):
		return LocaleEnUS
	case strings.HasPrefix(value, "ja"):
		return LocaleJaJP
	case strings.HasPrefix(value, "ko"):
		return LocaleKoKR
	case strings.HasPrefix(value, "es"):
		return LocaleEsES
	default:
		return LocaleZhCN
	}
}

func MarkAPI(c *gin.Context, locale string) {
	locale = NormalizeLocale(locale)
	c.Set(contextAPIKey, true)
	c.Set(contextLocaleKey, locale)
	c.Header("Content-Language", locale)
}

func IsAPI(c *gin.Context) bool {
	value, _ := c.Get(contextAPIKey)
	result, _ := value.(bool)
	return result
}

func Locale(c *gin.Context) string {
	value, _ := c.Get(contextLocaleKey)
	locale, _ := value.(string)
	return NormalizeLocale(locale)
}

func ErrorMessage(locale string, code, httpStatus int) string {
	key := errorKey(code, httpStatus)
	locale = NormalizeLocale(locale)
	if message := messages[locale][key]; message != "" {
		return message
	}
	return messages[LocaleZhCN][key]
}

func errorKey(code, httpStatus int) string {
	switch httpStatus {
	case http.StatusRequestEntityTooLarge:
		return "file_too_large"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	}
	switch code {
	case errcode.ErrParam:
		return "invalid_request"
	case errcode.ErrUnauthorized, errcode.ErrTokenInvalid, errcode.ErrTokenExpired, http.StatusUnauthorized:
		return "unauthorized"
	case errcode.ErrForbidden, http.StatusForbidden:
		return "forbidden"
	case errcode.ErrNotFound, errcode.ErrUserNotFound, errcode.ErrRoleNotFound, errcode.ErrMenuNotFound, http.StatusNotFound:
		return "not_found"
	default:
		return "server_error"
	}
}
