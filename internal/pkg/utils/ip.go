package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func ClientIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	ip = c.GetHeader("X-Real-IP")
	if ip != "" {
		return ip
	}

	return c.ClientIP()
}

type IPInfoResponse struct {
	Country string `json:"country"`
}

func GetCountryByIP(ip string) (string, error) {
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid ip: %s", ip)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	api := fmt.Sprintf("https://ipinfo.io/%s/json", url.PathEscape(ip))

	resp, err := client.Get(api)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result IPInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Country == "" {
		return "", fmt.Errorf("country not found")
	}

	return result.Country, nil
}
