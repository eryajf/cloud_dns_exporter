package export

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/eryajf/cloud_dns_exporter/public/logger"
	"github.com/weppos/publicsuffix-go/publicsuffix"

	"github.com/eryajf/cloud_dns_exporter/pkg/provider"
	"github.com/eryajf/cloud_dns_exporter/public"
	"github.com/robfig/cron/v3"
)

// InitCron 初始化定时任务
func InitCron() {
	c := cron.New(cron.WithSeconds())
	_, _ = c.AddFunc("*/30 * * * * *", func() {
		loading()
	})
	loading()
	_, _ = c.AddFunc("03 03 03 * * *", func() {
		loadingCert()
		loadingCustomRecordCert()
	})
	loadingCert()
	loadingCustomRecordCert()

	c.Start()
}

func loading() {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for cloudProvider, accounts := range public.Config.CloudProviders {
		for _, cloudAccount := range accounts.Accounts {
			wg.Add(1)
			go func(cloudProvider, cloudName string, account map[string]string) {
				defer wg.Done()
				domainListCacheKey := public.DomainList + "_" + cloudProvider + "_" + cloudName
				dnsProvider, err := provider.Factory.Create(cloudProvider, account)
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] create provider failed: %v", domainListCacheKey, err))
					return
				}
				domains, err := dnsProvider.ListDomains()
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] list domains failed: %v", domainListCacheKey, err))
					return
				}

				mu.Lock()
				value, err := json.Marshal(domains)
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] marshal domain list failed: %v", domainListCacheKey, err))
				}
				if err := public.Cache.Set(domainListCacheKey, value); err != nil {
					logger.Error(fmt.Sprintf("[ %s ] cache domain list failed: %v", domainListCacheKey, err))
				}
				mu.Unlock()

				recordListCacheKey := public.RecordList + "_" + cloudProvider + "_" + cloudName
				records, err := dnsProvider.ListRecords()
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] list records failed: %v", recordListCacheKey, err))
					return
				}
				mu.Lock()
				value, err = json.Marshal(records)
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] marshal record list failed: %v", recordListCacheKey, err))
				}
				if err := public.Cache.Set(recordListCacheKey, value); err != nil {
					logger.Error(fmt.Sprintf("[ %s ] cache record list failed: %v", recordListCacheKey, err))
				}
				mu.Unlock()
			}(cloudProvider, cloudAccount["name"], cloudAccount)
		}
	}
	wg.Wait()
}

func loadingCert() {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for cloudProvider, accounts := range public.Config.CloudProviders {
		for _, cloudAccount := range accounts.Accounts {
			wg.Add(1)
			go func(cloudProvider, cloudName string, account map[string]string) {
				defer wg.Done()
				recordListCacheKey := public.RecordList + "_" + cloudProvider + "_" + cloudName
				var records []provider.Record
				rst2, err := public.Cache.Get(recordListCacheKey)
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] get record list failed: %v", recordListCacheKey, err))
				}
				err = json.Unmarshal(rst2, &records)
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] json.Unmarshal error: %v", recordListCacheKey, err))
				}
				var recordCertReq []provider.GetRecordCertReq
				for _, v := range getNewRecord(records) {
					recordCertReq = append(recordCertReq, provider.GetRecordCertReq{
						CloudProvider: v.CloudProvider,
						CloudName:     v.CloudName,
						DomainName:    v.DomainName,
						FullRecord:    v.FullRecord,
						RecordValue:   v.RecordValue,
						RecordID:      v.RecordID,
					})
				}
				recordCerts, err := GetMultipleCertInfo(recordCertReq)
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] get record cert info failed: %v", recordListCacheKey, err))
					return
				}

				mu.Lock()
				recordCertInfoCacheKey := public.RecordCertInfo + "_" + cloudProvider + "_" + cloudName
				value, err := json.Marshal(recordCerts)
				if err != nil {
					logger.Error(fmt.Sprintf("[ %s ] marshal domain list failed: %v", recordCertInfoCacheKey, err))
				}
				if err := public.CertCache.Set(recordCertInfoCacheKey, value); err != nil {
					logger.Error(fmt.Sprintf("[ %s ] cache domain list failed: %v", recordCertInfoCacheKey, err))
				}
				mu.Unlock()
			}(cloudProvider, cloudAccount["name"], cloudAccount)
		}
	}
	wg.Wait()
}

func loadingCustomRecordCert() {
	if len(public.Config.CustomRecords) == 0 {
		return
	}
	var records []provider.Record
	for _, v := range public.Config.CustomRecords {
		domainName, err := publicsuffix.Domain(v)
		if err != nil {
			logger.Error(fmt.Sprintf("[ custom ] get domain failed: %v", err))
		}
		records = append(records, provider.Record{
			CloudProvider: public.CustomRecords,
			CloudName:     public.CustomRecords,
			DomainName:    domainName,
			FullRecord:    v,
			RecordValue:   v,
			RecordID:      public.GetID(),
			RecordType:    "CNAME", // 默认指定为CNAME记录,这两条记录为了通过检测
			RecordStatus:  "enable",
		})
	}
	var recordCertReq []provider.GetRecordCertReq
	for _, v := range getNewRecord(records) {
		recordCertReq = append(recordCertReq, provider.GetRecordCertReq{
			CloudProvider: v.CloudProvider,
			CloudName:     v.CloudName,
			DomainName:    v.DomainName,
			FullRecord:    v.FullRecord,
			RecordValue:   v.RecordValue,
			RecordID:      v.RecordID,
		})
	}
	recordCerts, err := GetMultipleCertInfo(recordCertReq)
	if err != nil {
		logger.Error(fmt.Sprintf("[ custom ] get record cert info failed: %v", err))
		return
	}
	recordCertInfoCacheKey := public.RecordCertInfo + "_" + public.CustomRecords
	value, err := json.Marshal(recordCerts)
	if err != nil {
		logger.Error(fmt.Sprintf("[ %s ] marshal domain list failed: %v", recordCertInfoCacheKey, err))
	}
	if err := public.CertCache.Set(recordCertInfoCacheKey, value); err != nil {
		logger.Error(fmt.Sprintf("[ %s ] cache domain list failed: %v", recordCertInfoCacheKey, err))
	}
}
