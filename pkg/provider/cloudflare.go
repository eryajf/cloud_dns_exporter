package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/cloudflare/cloudflare-go"
	"github.com/eryajf/cloud_dns_exporter/public"
	"github.com/golang-module/carbon/v2"
	"sync"
	"time"
)

type CloudFlareDNS struct {
	account public.Account
	client  *cloudflare.API
}

type Header struct {
	XAuthEmail  interface{} `json:"X-Auth-Email"`
	XAuthKey    interface{} `json:"X-Auth-Key"`
	ContentType string      `json:"Content-Type"`
}

func NewCloudflareDNSClient(token string, email string) (*cloudflare.API, error) {
	return cloudflare.New(token, email)
}

func NewCloudFlareDNS(account public.Account) *CloudFlareDNS {
	client, _ := NewCloudflareDNSClient(account.SecretKey, account.SecretID)
	return &CloudFlareDNS{
		account: account,
		client:  client,
	}
}

func (cf *CloudFlareDNS) ListDomains() ([]Domain, error) {
	cfd := NewCloudFlareDNS(public.Account{
		CloudProvider: cf.account.CloudProvider,
		CloudName:     cf.account.CloudName,
		SecretID:      cf.account.SecretID,
		SecretKey:     cf.account.SecretKey,
	})
	cf.client = cfd.client
	var (
		dataObj []Domain
		wg      sync.WaitGroup
		mu      sync.Mutex
	)
	domains, err := cf.getDomainList()
	if err != nil {
		return nil, err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	for _, domain := range domains {
		wg.Add(1)
		go func(domain cloudflare.Zone) {
			defer wg.Done()
			<-ticker.C
			domainCreateAndExpiryDate, _ := cf.getDomainCreateAndExpiryDate(domain)
			mu.Lock()
			dataObj = append(dataObj, Domain{
				CloudName:       domain.Name,
				CloudProvider:   cf.account.CloudProvider,
				CreatedDate:     domainCreateAndExpiryDate.CreatedDate,
				DaysUntilExpiry: domainCreateAndExpiryDate.DaysUntilExpiry,
				DomainID:        domain.ID,
				DomainName:      domain.Name,
				DomainRemark:    tea.StringValue(nil),
				DomainStatus:    domain.Status,
				ExpiryDate:      domainCreateAndExpiryDate.ExpiryDate,
			})
			mu.Unlock()
		}(domain)
	}
	wg.Wait()
	return dataObj, err
}

func (cf *CloudFlareDNS) ListRecords() ([]Record, error) {
	var (
		dataObj []Record
		domains []Domain
		wg      sync.WaitGroup
		mu      sync.Mutex
	)
	cfd := NewCloudFlareDNS(public.Account{
		CloudProvider: cf.account.CloudProvider,
		CloudName:     cf.account.CloudName,
		SecretID:      cf.account.SecretID,
		SecretKey:     cf.account.SecretKey,
	})
	cf.client = cfd.client
	rst, err := public.Cache.Get(public.DomainList + "_" + cf.account.CloudProvider + "_" + cf.account.CloudName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rst, &domains)
	if err != nil {
		return nil, err
	}
	results := make(map[string][]cloudflare.DNSRecord)
	ticker := time.NewTicker(100 * time.Millisecond)
	for _, domain := range domains {
		wg.Add(1)
		go func(domain Domain) {
			defer wg.Done()
			<-ticker.C
			records, err := cf.getRecordList(domain.DomainName)
			if err != nil {
				fmt.Printf("cloudflare get record list error: %v", err)
				return
			}
			mu.Lock()
			results[domain.DomainName] = records
			mu.Unlock()
		}(domain)
	}
	wg.Wait()
	for domain, records := range results {
		for _, record := range records {
			dataObj = append(dataObj, Record{
				CloudName:     cf.account.CloudName,
				CloudProvider: cf.account.CloudProvider,
				DomainName:    domain,
				RecordID:      record.ID,
				RecordName:    record.Name,
				RecordType:    record.Type,
				RecordRemark:  tea.StringValue(nil),
				RecordStatus:  "enable",
				RecordTTL:     fmt.Sprintf("%d", record.TTL),
				FullRecord:    record.Name,
			})
		}
	}
	return dataObj, err
}

// getDomainList 获取解析域域名列表
func (cf *CloudFlareDNS) getDomainList() (rst []cloudflare.Zone, err error) {
	client, err := NewCloudflareDNSClient(cf.account.SecretKey, cf.account.SecretID)
	if err != nil {
		fmt.Printf("cloudflare client init error: %v", err)
		return
	}
	zones, err := client.ListZones(context.Background())
	if err != nil {
		fmt.Printf("cloudflare list zones error: %v", err)
		return
	}
	for _, zone := range zones {
		rst = append(rst, zone)
	}
	return
}

func (cf *CloudFlareDNS) getAccountId() (account cloudflare.Account, err error) {
	client, _ := NewCloudflareDNSClient(cf.account.SecretKey, cf.account.SecretID)
	accounts, _, err := client.Accounts(context.Background(), cloudflare.AccountsListParams{})
	if err != nil {
		return
	}
	for _, a := range accounts {
		account = a
	}
	return
}

func (cf *CloudFlareDNS) getRecordList(domain string) (rst []cloudflare.DNSRecord, err error) {
	page := 1
	pageSize := 2
	client, _ := NewCloudflareDNSClient(cf.account.SecretKey, cf.account.SecretID)
	zoneID, err := client.ZoneIDByName(domain)
	if err != nil {
		return
	}
	for {
		records, r, err := client.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
			ResultInfo: cloudflare.ResultInfo{Page: page, PerPage: pageSize},
		})
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			rst = append(rst, record)
		}
		if page*pageSize > r.Total {
			break
		}
		page++
	}
	return
}

func (cf *CloudFlareDNS) getDomainCreateAndExpiryDate(domain cloudflare.Zone) (d Domain, err error) {
	client, err := NewCloudflareDNSClient(cf.account.SecretKey, cf.account.SecretID)
	if err != nil {
		return
	}
	account, err := cf.getAccountId()
	if err != nil {
		return
	}
	domainInfo, err := client.RegistrarDomain(context.Background(), account.ID, domain.Name)
	d.CreatedDate = domainInfo.CreatedAt.String()
	d.ExpiryDate = domainInfo.ExpiresAt.String()
	if d.ExpiryDate != "" {
		d.DaysUntilExpiry = carbon.Now().DiffInDays(carbon.Parse(d.ExpiryDate))
	}
	return
}
