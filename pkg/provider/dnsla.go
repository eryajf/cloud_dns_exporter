package provider

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/eryajf/cloud_dns_exporter/dnslib/dnsla"
	"github.com/eryajf/cloud_dns_exporter/public"
	"github.com/eryajf/cloud_dns_exporter/public/logger"
	"github.com/golang-module/carbon/v2"
)

type DNSLaDNS struct {
	account public.Account
	client  *dnsla.Client
}

// NewDNSLaClient 初始化客户端
func NewDNSLaClient(secretID, secretKey string) (*dnsla.Client, error) {
	client, err := dnsla.NewClient(secretID, secretKey)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewDNSLaDNS 创建 DNSLaDNS 实例
func NewDNSLaDNS(account public.Account) (*DNSLaDNS, error) {
	client, err := NewDNSLaClient(account.SecretID, account.SecretKey)
	if err != nil {
		return nil, err
	}
	return &DNSLaDNS{
		account: account,
		client:  client,
	}, nil
}

// ListDomains 获取域名列表
func (d *DNSLaDNS) ListDomains() ([]Domain, error) {
	gd, err := NewDNSLaDNS(public.Account{
		CloudProvider: d.account.CloudProvider,
		CloudName:     d.account.CloudName,
		SecretID:      d.account.SecretID,
		SecretKey:     d.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	d.client = gd.client
	var dataObj []Domain
	domains, err := d.getDomainList()
	if err != nil {
		return nil, err
	}
	for _, v := range domains {
		dataObj = append(dataObj, Domain{
			CloudProvider:   d.account.CloudProvider,
			CloudName:       d.account.CloudName,
			DomainID:        v.ID,
			DomainName:      v.Domain,
			DomainRemark:    v.Domain,
			DomainStatus:    oneStatus(strconv.Itoa(v.State)),
			CreatedDate:     carbon.CreateFromTimestampMilli(v.CreatedAt).ToDateTimeString(),
			ExpiryDate:      carbon.CreateFromTimestampMilli(v.ExpiredAt).ToDateTimeString(),
			DaysUntilExpiry: carbon.Now().DiffInDays(carbon.Parse(carbon.CreateFromTimestampMilli(v.ExpiredAt).ToDateTimeString())),
		})
	}
	return dataObj, nil
}

// ListRecords 获取记录列表
func (d *DNSLaDNS) ListRecords() ([]Record, error) {
	var (
		dataObj []Record
		domains []Domain
		wg      sync.WaitGroup
		mu      sync.Mutex
	)
	tcd, err := NewDNSLaDNS(public.Account{
		CloudProvider: d.account.CloudProvider,
		CloudName:     d.account.CloudName,
		SecretID:      d.account.SecretID,
		SecretKey:     d.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	d.client = tcd.client
	rst, err := public.Cache.Get(public.DomainList + "_" + d.account.CloudProvider + "_" + d.account.CloudName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rst, &domains)
	if err != nil {
		return nil, err
	}
	results := make(map[string][]dnsla.Record)
	ticker := time.NewTicker(time.Second)
	for _, domain := range domains {
		wg.Add(1)
		go func(domainName, domainId string) {
			defer wg.Done()
			<-ticker.C
			records, err := d.getRecordList(domainId)
			if err != nil {
				logger.Error(fmt.Sprintf("[ %s_%s ] get record list failed: %v", d.account.CloudProvider, d.account.CloudName, err))
			}
			if len(records) == 0 {
				return
			}
			mu.Lock()
			results[domainName] = records
			mu.Unlock()
		}(domain.DomainName, domain.DomainID)
	}
	wg.Wait()
	for domain, records := range results {
		for _, v := range records {
			dataObj = append(dataObj, Record{
				CloudProvider: d.account.CloudProvider,
				CloudName:     d.account.CloudName,
				DomainName:    domain,
				RecordID:      v.Data,
				RecordType:    getRecordType(v.Type),
				RecordName:    v.DisplayHost,
				RecordValue:   v.Data,
				RecordTTL:     strconv.Itoa(v.TTL),
				RecordWeight:  strconv.Itoa(v.Weight),
				RecordStatus:  "enable",
				RecordRemark:  v.DisplayHost,
				FullRecord:    v.DisplayHost + "." + domain,
			})
		}
	}
	return dataObj, nil
}

// https://www.dns.la/docs/ApiDoc
// GetDomainList 获取云解析中域名列表
func (d *DNSLaDNS) getDomainList() ([]dnsla.Domain, error) {
	domains, err := d.client.Domains.List(dnsla.NewPageOption(1, 500))
	if err != nil {
		return nil, err
	}

	return domains.Data.Results, nil
}

// https://www.dns.la/docs/ApiDoc
// RecordList 域名记录列表
func (d *DNSLaDNS) getRecordList(domain string) ([]dnsla.Record, error) {
	// TODO 目前写死的获取1000条记录
	rds, err := d.client.Records.List(dnsla.NewPageOption(1, 1000), domain)
	if err != nil {
		fmt.Printf("Error listing records: %v\n", err)
	}
	return rds.Data.Results, err
}

// RecordType 表示DNS记录类型
type RecordType int

// 定义常量
const (
	A          RecordType = 1
	NS         RecordType = 2
	CNAME      RecordType = 5
	MX         RecordType = 15
	TXT        RecordType = 16
	AAAA       RecordType = 28
	SRV        RecordType = 33
	CAA        RecordType = 257
	URLForward RecordType = 256
)

// getRecordType 将数字类型转换为对应的记录类型字符串
func getRecordType(typeNum int) string {
	switch RecordType(typeNum) {
	case A:
		return "A"
	case NS:
		return "NS"
	case CNAME:
		return "CNAME"
	case MX:
		return "MX"
	case TXT:
		return "TXT"
	case AAAA:
		return "AAAA"
	case SRV:
		return "SRV"
	case CAA:
		return "CAA"
	case URLForward:
		return "URL转发"
	default:
		return "Unknown"
	}
}
