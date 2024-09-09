package dnsla

import (
	"fmt"
	"net/url"
	"strconv"
)

// DomainService 域名服务
type DomainService struct{ *Client }

// List 获取域名列表
func (d *DomainService) List(page PageOption, options ...DomainListOption) (*DomainListResponse, error) {
	params := url.Values{}
	params.Set("pageIndex", strconv.Itoa(page.PageIndex))
	params.Set("pageSize", strconv.Itoa(page.PageSize))
	for _, option := range options {
		option(params)
	}
	resp, err := d.client.R().
		SetQueryParamsFromValues(params).
		SetResult(&DomainListResponse{}).
		Get("/api/domainList")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*DomainListResponse)
	if !ok {
		return nil, fmt.Errorf("failed to cast response to *Response")
	}

	return result, nil
}
