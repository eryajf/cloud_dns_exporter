package provider

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/eryajf/cloud_dns_exporter/public/logger"
	"github.com/golang-module/carbon/v2"

	"github.com/alyx/go-daddy/daddy"
	"github.com/eryajf/cloud_dns_exporter/public"
)

type GodaddyDNS struct {
	account public.Account
	client  *daddy.Client
}

// NewGodaddyClient 初始化客户端
func NewGodaddyClient(secretID, secretKey string) (*daddy.Client, error) {
	client, err := daddy.NewClient(secretID, secretKey, false)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewGodaddyDNS 创建 GodaddyDNS 实例
func NewGodaddyDNS(account public.Account) (*GodaddyDNS, error) {
	client, err := NewGodaddyClient(account.SecretID, account.SecretKey)
	if err != nil {
		return nil, err
	}
	return &GodaddyDNS{
		account: account,
		client:  client,
	}, nil
}

// ListDomains 获取域名列表
func (g *GodaddyDNS) ListDomains() ([]Domain, error) {
	gd, err := NewGodaddyDNS(public.Account{
		CloudProvider: g.account.CloudProvider,
		CloudName:     g.account.CloudName,
		SecretID:      g.account.SecretID,
		SecretKey:     g.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	g.client = gd.client
	var dataObj []Domain
	domains, err := g.getDomainList()
	if err != nil {
		return nil, err
	}
	for _, v := range domains {
		dataObj = append(dataObj, Domain{
			CloudProvider:   g.account.CloudProvider,
			CloudName:       g.account.CloudName,
			DomainID:        strconv.Itoa(v.DomainID),
			DomainName:      v.Domain,
			DomainRemark:    v.Domain,
			DomainStatus:    oneStatus(v.Status),
			CreatedDate:     v.CreatedAt,
			ExpiryDate:      v.Expires,
			DaysUntilExpiry: carbon.Now().DiffInDays(carbon.Parse(v.Expires)),
		})
	}
	return dataObj, nil
}

// ListRecords 获取记录列表
func (g *GodaddyDNS) ListRecords() ([]Record, error) {
	var (
		dataObj []Record
		domains []Domain
		wg      sync.WaitGroup
		mu      sync.Mutex
	)
	tcd, err := NewGodaddyDNS(public.Account{
		CloudProvider: g.account.CloudProvider,
		CloudName:     g.account.CloudName,
		SecretID:      g.account.SecretID,
		SecretKey:     g.account.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	g.client = tcd.client
	rst, err := public.Cache.Get(public.DomainList + "_" + g.account.CloudProvider + "_" + g.account.CloudName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rst, &domains)
	if err != nil {
		return nil, err
	}
	results := make(map[string][]daddy.DNSRecord)
	ticker := time.NewTicker(time.Second)
	for _, domain := range domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			<-ticker.C
			records, err := g.getRecordList(domain)
			if err != nil {
				logger.Error(fmt.Sprintf("[ %s_%s ] get record list failed: %v", g.account.CloudProvider, g.account.CloudName, err))
			}
			if len(records) == 0 {
				return
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
				CloudProvider: g.account.CloudProvider,
				CloudName:     g.account.CloudName,
				DomainName:    domain,
				RecordID:      v.Data,
				RecordType:    v.Type,
				RecordName:    v.Name,
				RecordValue:   v.Data,
				RecordTTL:     strconv.Itoa(v.TTL),
				RecordWeight:  strconv.Itoa(v.Weight),
				RecordStatus:  "enable",
				RecordRemark:  v.Name,
				FullRecord:    v.Name + "." + domain,
			})
		}
	}
	return dataObj, nil
}

// https://developer.godaddy.com/doc/endpoint/domains
// GetDomainList 获取云解析中域名列表
func (g *GodaddyDNS) getDomainList() ([]daddy.DomainSummary, error) {
	domains, err := g.client.Domains.List(nil, nil, 0, "", nil, "")
	if err != nil {
		return nil, err
	}

	return domains, nil
}

// https://developer.godaddy.com/doc/endpoint/domains
// RecordList 域名记录列表
func (g *GodaddyDNS) getRecordList(domain string) ([]daddy.DNSRecord, error) {
	// TODO 目前写死的获取500条记录
	rds, err := g.client.Domains.GetRecords(domain, "", "", 0, 500)
	if err != nil {
		fmt.Printf("Error listing records: %v\n", err)
	}
	return rds, err
}
