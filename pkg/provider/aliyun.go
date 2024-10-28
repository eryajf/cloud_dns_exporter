package provider

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	alidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	domain "github.com/alibabacloud-go/domain-20180129/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/golang-module/carbon/v2"

	"github.com/eryajf/cloud_dns_exporter/public/logger"

	"github.com/eryajf/cloud_dns_exporter/public"
)

type AliyunDNS struct {
	account public.Account
	client  *alidns.Client
}

// NewAliyunClient 初始化客户端
func NewAliyunClient(secretID, secretKey string) (*alidns.Client, error) {
	config := openapi.Config{
		AccessKeyId:     tea.String(secretID),
		AccessKeySecret: tea.String(secretKey),
	}
	config.Endpoint = tea.String("dns.aliyuncs.com")
	client, err := alidns.NewClient(&config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewAliyunDNS 创建实例
func NewAliyunDNS(account public.Account) (*AliyunDNS, error) {
	client, err := NewAliyunClient(account.SecretID, account.SecretKey)
	if err != nil {
		return nil, err
	}
	return &AliyunDNS{
		account: account,
		client:  client,
	}, nil
}

// ListDomains 获取域名列表
func (a *AliyunDNS) ListDomains() ([]Domain, error) {
	tcd, err := NewAliyunDNS(public.Account{
		CloudProvider: a.account.CloudProvider,
		CloudName:     a.account.CloudName,
		SecretID:      a.account.SecretID,
		SecretKey:     a.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	a.client = tcd.client

	var dataObj []Domain
	domains, err := a.getDomainList()
	if err != nil {
		return nil, err
	}
	domainNames, err := a.getDomainNameList()
	if err != nil {
		return nil, err
	}
	for _, v := range domains {
		domainCreateAndExpiryDate := a.getDomainCreateAndExpiryDate(domainNames, v)
		dataObj = append(dataObj, Domain{
			CloudProvider:   a.account.CloudProvider,
			CloudName:       a.account.CloudName,
			DomainID:        tea.StringValue(v.DomainId),
			DomainName:      tea.StringValue(v.DomainName),
			DomainRemark:    tea.StringValue(v.Remark),
			DomainStatus:    "enable",
			CreatedDate:     domainCreateAndExpiryDate.CreatedDate,
			ExpiryDate:      domainCreateAndExpiryDate.ExpiryDate,
			DaysUntilExpiry: domainCreateAndExpiryDate.DaysUntilExpiry,
		})
	}
	return dataObj, nil
}

// ListRecords 获取记录列表
func (a *AliyunDNS) ListRecords() ([]Record, error) {
	var (
		dataObj []Record
		domains []Domain
		wg      sync.WaitGroup
		mu      sync.Mutex
	)
	tcd, err := NewAliyunDNS(public.Account{
		CloudProvider: a.account.CloudProvider,
		CloudName:     a.account.CloudName,
		SecretID:      a.account.SecretID,
		SecretKey:     a.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	a.client = tcd.client
	rst, err := public.Cache.Get(public.DomainList + "_" + a.account.CloudProvider + "_" + a.account.CloudName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rst, &domains)
	if err != nil {
		return nil, err
	}
	results := make(map[string][]*alidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for _, domain := range domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			<-ticker.C
			records, err := a.getRecordList(domain)
			if err != nil {
				logger.Error("get record list failed: %v", err)
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
				CloudProvider: a.account.CloudProvider,
				CloudName:     a.account.CloudName,
				DomainName:    domain,
				RecordID:      tea.StringValue(v.RecordId),
				RecordType:    tea.StringValue(v.Type),
				RecordName:    tea.StringValue(v.RR),
				RecordValue:   tea.StringValue(v.Value),
				RecordTTL:     fmt.Sprintf("%d", tea.Int64Value(v.TTL)),
				RecordWeight:  fmt.Sprintf("%d", tea.Int32Value(v.Weight)),
				RecordStatus:  oneStatus(tea.StringValue(v.Status)),
				RecordRemark:  tea.StringValue(v.Remark),
				UpdateTime:    carbon.CreateFromTimestampMilli(tea.Int64Value(v.UpdateTimestamp)).ToDateTimeString(),
				FullRecord:    tea.StringValue(v.RR) + "." + domain,
			})
		}
	}
	return dataObj, nil
}

// https://next.api.aliyun.com/document/Alidns/2015-01-09/DescribeDomains
// GetDomains 获取域名列表
func (a *AliyunDNS) getDomainList() (rst []*alidns.DescribeDomainsResponseBodyDomainsDomain, err error) {
	pageNumber := int64(1)
	pageSize := int64(100)
	for {
		resp, err := a.client.DescribeDomains(&alidns.DescribeDomainsRequest{
			PageNumber: tea.Int64(pageNumber),
			PageSize:   tea.Int64(pageSize),
		})
		if err != nil {
			return nil, err
		}
		rst = append(rst, resp.Body.Domains.Domain...)
		if len(resp.Body.Domains.Domain) < int(pageSize) {
			break
		}
		pageNumber++
	}
	return
}

// https://next.api.aliyun.com/document/Alidns/2015-01-09/DescribeDomainRecords
// GetDomainList 获取记录列表
func (a *AliyunDNS) getRecordList(domain string) (rst []*alidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord, err error) {
	var (
		pageNumber int64 = 1
		pageSize   int64 = 500
	)
	for {
		resp, err := a.client.DescribeDomainRecords(&alidns.DescribeDomainRecordsRequest{
			DomainName: tea.String(domain),
			PageNumber: tea.Int64(pageNumber),
			PageSize:   tea.Int64(pageSize),
		})
		if err != nil {
			return nil, err
		}
		rst = append(rst, resp.Body.DomainRecords.Record...)
		if len(resp.Body.DomainRecords.Record) < int(pageSize) {
			break
		}
		pageNumber++
	}
	return
}

// https://next.api.aliyun.com/document/Domain/2018-01-29/QueryDomainList
// getDomainNameList 获取域名列表
func (a *AliyunDNS) getDomainNameList() (rst []*domain.QueryDomainListResponseBodyDataDomain, err error) {
	config := openapi.Config{
		AccessKeyId:     tea.String(a.account.SecretID),
		AccessKeySecret: tea.String(a.account.SecretKey),
	}
	config.Endpoint = tea.String("domain.aliyuncs.com")
	client, err := domain.NewClient(&config)
	if err != nil {
		return nil, err
	}
	var (
		pageNumber int32 = 1
		pageSize   int32 = 500
	)
	for {
		resp, err := client.QueryDomainList(&domain.QueryDomainListRequest{
			PageNum:  tea.Int32(pageNumber),
			PageSize: tea.Int32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		rst = append(rst, resp.Body.Data.Domain...)
		if len(resp.Body.Data.Domain) < int(pageSize) {
			break
		}
		pageNumber++
	}
	return
}

// getDomainCreateAndExpiryDate 获取域名的创建时间与到期时间
func (a *AliyunDNS) getDomainCreateAndExpiryDate(domainList []*domain.QueryDomainListResponseBodyDataDomain, domain *alidns.DescribeDomainsResponseBodyDomainsDomain) (d Domain) {
	for _, v := range domainList {
		if tea.StringValue(v.DomainName) == tea.StringValue(domain.DomainName) {
			d.CreatedDate = tea.StringValue(v.RegistrationDate)
			d.ExpiryDate = tea.StringValue(v.ExpirationDate)
			if d.ExpiryDate != "" {
				d.DaysUntilExpiry = carbon.Now().DiffInDays(carbon.Parse(d.ExpiryDate))
			}
		}
	}
	return
}
