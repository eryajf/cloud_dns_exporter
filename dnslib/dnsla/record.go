package dnsla

import (
	"fmt"
	"net/url"
	"strconv"
)

// RecordService 解析记录服务
type RecordService struct{ *Client }

// ListRecords 获取域名解析记录列表
func (r *RecordService) List(page PageOption, domainID string, options ...RecordListOption) (*RecordListResponse, error) {
	params := url.Values{}
	params.Set("pageIndex", strconv.Itoa(page.PageIndex))
	params.Set("pageSize", strconv.Itoa(page.PageSize))
	params.Set("domainId", domainID)
	for _, option := range options {
		option(params)
	}
	resp, err := r.client.R().
		SetQueryParamsFromValues(params).
		SetResult(&RecordListResponse{}).
		Get("/api/recordList")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode(), resp.String())
	}
	result, ok := resp.Result().(*RecordListResponse)
	if !ok {
		return nil, fmt.Errorf("failed to cast response to *RecordResponse")
	}
	return result, nil
}
