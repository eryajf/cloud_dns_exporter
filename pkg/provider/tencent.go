package provider

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/eryajf/cloud_dns_exporter/public/logger"

	"github.com/eryajf/cloud_dns_exporter/public"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

type TencentCloudDNS struct {
	account public.Account
	client  *dnspod.Client
}

// NewTencentClient 初始化客户端
func NewTencentClient(secretID, secretKey string) (*dnspod.Client, error) {
	credential := common.NewCredential(secretID, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	client, err := dnspod.NewClient(credential, "", cpf)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewTencentCloudDNS 创建 TencentCloudDNS 实例
func NewTencentCloudDNS(account public.Account) (*TencentCloudDNS, error) {
	client, err := NewTencentClient(account.SecretID, account.SecretKey)
	if err != nil {
		return nil, err
	}
	return &TencentCloudDNS{
		account: account,
		client:  client,
	}, nil
}

// ListDomains 获取域名列表
func (t *TencentCloudDNS) ListDomains() ([]Domain, error) {
	tcd, err := NewTencentCloudDNS(public.Account{
		CloudProvider: t.account.CloudProvider,
		CloudName:     t.account.CloudName,
		SecretID:      t.account.SecretID,
		SecretKey:     t.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	t.client = tcd.client

	var dataObj []Domain
	domains, err := t.getDomainList()
	if err != nil {
		return nil, err
	}
	for _, v := range domains {
		dataObj = append(dataObj, Domain{
			CloudProvider: t.account.CloudProvider,
			CloudName:     t.account.CloudName,
			DomainID:      fmt.Sprintf("%d", tea.Uint64Value(v.DomainId)),
			DomainName:    tea.StringValue(v.Name),
			DomainRemark:  tea.StringValue(v.Remark),
			DomainStatus:  oneStatus(tea.StringValue(v.Status)),
			CreateTime:    tea.StringValue(v.CreatedOn),
		})
	}
	return dataObj, nil
}

// ListRecords 获取记录列表
func (t *TencentCloudDNS) ListRecords() ([]Record, error) {
	var (
		dataObj []Record
		domains []Domain
		wg      sync.WaitGroup
		mu      sync.Mutex
	)
	tcd, err := NewTencentCloudDNS(public.Account{
		CloudProvider: t.account.CloudProvider,
		CloudName:     t.account.CloudName,
		SecretID:      t.account.SecretID,
		SecretKey:     t.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	t.client = tcd.client
	rst, err := public.Cache.Get(public.DomainList + "_" + t.account.CloudProvider + "_" + t.account.CloudName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rst, &domains)
	if err != nil {
		return nil, err
	}
	results := make(map[string][]*dnspod.RecordListItem)
	ticker := time.NewTicker(100 * time.Millisecond)
	for _, domain := range domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			<-ticker.C
			records, err := t.getRecordList(domain)
			if err != nil {
				logger.Error(fmt.Sprintf("[ %s_%s ] get record list failed: %v", t.account.CloudProvider, t.account.CloudName, err))
			}
			mu.Lock()
			results[domain] = records
			mu.Unlock()
		}(domain.DomainName)
	}
	wg.Wait()
	for domain, records := range results {
		for _, v := range records {
			dataObj = append(dataObj, Record{
				CloudProvider: t.account.CloudProvider,
				CloudName:     t.account.CloudName,
				DomainName:    domain,
				RecordID:      fmt.Sprintf("%d", tea.Uint64Value(v.RecordId)),
				RecordType:    tea.StringValue(v.Type),
				RecordName:    tea.StringValue(v.Name),
				RecordValue:   tea.StringValue(v.Value),
				RecordTTL:     fmt.Sprintf("%d", tea.Uint64Value(v.TTL)),
				RecordWeight:  fmt.Sprintf("%d", tea.Uint64Value(v.Weight)),
				RecordStatus:  oneStatus(tea.StringValue(v.Status)),
				RecordRemark:  tea.StringValue(v.Remark),
				UpdateTime:    tea.StringValue(v.UpdatedOn),
				FullRecord:    tea.StringValue(v.Name) + "." + domain,
			})
		}
	}
	return dataObj, nil
}

// https://cloud.tencent.com/document/api/1427/56172
// GetDomainList 获取云账号下域名列表
func (t *TencentCloudDNS) getDomainList() ([]*dnspod.DomainListItem, error) {
	request := dnspod.NewDescribeDomainListRequest()
	response, err := t.client.DescribeDomainList(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return response.Response.DomainList, nil
}

// https://cloud.tencent.com/document/api/1427/56166
// RecordList 域名记录列表
func (t *TencentCloudDNS) getRecordList(domain string) ([]*dnspod.RecordListItem, error) {
	var (
		offset uint64 = 0
		limit  uint64 = 3000
		temp   []*dnspod.RecordListItem
	)
	request := dnspod.NewDescribeRecordListRequest()
	request.Domain = common.StringPtr(domain)
	for {
		request.Offset = common.Uint64Ptr(offset)
		request.Limit = common.Uint64Ptr(limit)
		response, err := t.client.DescribeRecordList(request)
		if e, ok := err.(*errors.TencentCloudSDKError); ok {
			if e.Code == "ResourceNotFound.NoDataOfRecord" {
				return temp, nil
			}
		}
		if err != nil {
			return temp, err
		}
		temp = append(temp, response.Response.RecordList...)
		if len(response.Response.RecordList) == 0 {
			break
		}
		offset += limit
	}
	return temp, nil
}
