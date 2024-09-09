package dnsla

import (
	"errors"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client DNS.LA 客户端
type Client struct {
	client *resty.Client

	// Services
	Domains *DomainService
	Records *RecordService
}

var baseUrl = "https://api.dns.la"

// NewClient 初始化客户端
func NewClient(key, secret string) (*Client, error) {
	c := new(Client)
	if key == "" {
		return c, errors.New("missing dns.la API key")
	}
	if secret == "" {
		return c, errors.New("missing dns.la API secret")
	}
	c.client = resty.New().SetBaseURL(baseUrl).SetBasicAuth(key, secret).
		SetTimeout(3 * time.Second).SetRetryCount(3).SetRetryWaitTime(2 * time.Second)
	// Initialize services
	c.Domains = &DomainService{c}
	c.Records = &RecordService{c}

	return c, nil
}
