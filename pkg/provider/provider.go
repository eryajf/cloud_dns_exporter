package provider

import (
	"fmt"
	"strings"

	"github.com/eryajf/cloud_dns_exporter/public"
)

var Factory *DNSProviderFactory

func init() {
	Factory = NewDNSProviderFactory()
	// 如有新的类型，则需要在此处注册，注册之后会自动识别并执行
	Factory.Register(public.TencentDnsProvider, func(account map[string]string) DNSProvider {
		return &TencentCloudDNS{
			account: public.Account{
				CloudProvider: public.TencentDnsProvider,
				CloudName:     account["name"],
				SecretID:      account["secretId"],
				SecretKey:     account["secretKey"],
			},
		}
	})
	Factory.Register(public.AliyunDnsProvider, func(account map[string]string) DNSProvider {
		return &AliyunDNS{
			account: public.Account{
				CloudProvider: public.AliyunDnsProvider,
				CloudName:     account["name"],
				SecretID:      account["secretId"],
				SecretKey:     account["secretKey"],
			},
		}
	})
	Factory.Register(public.GodaddyDnsProvider, func(account map[string]string) DNSProvider {
		return &GodaddyDNS{
			account: public.Account{
				CloudProvider: public.GodaddyDnsProvider,
				CloudName:     account["name"],
				SecretID:      account["secretId"],
				SecretKey:     account["secretKey"],
			},
		}
	})
	Factory.Register(public.AmazonDnsProvider, func(account map[string]string) DNSProvider {
		return &AmazonDNS{
			account: public.Account{
				CloudProvider: public.AmazonDnsProvider,
				CloudName:     account["name"],
				SecretID:      account["secretId"],
				SecretKey:     account["secretKey"],
			},
		}
	})
}

// Doamin 域名信息
type Domain struct {
	CloudProvider   string `json:"cloud_provider"`
	CloudName       string `json:"cloud_name"`
	DomainID        string `json:"domain_id"`
	DomainName      string `json:"domain_name"`
	DomainRemark    string `json:"domain_remark"`
	DomainStatus    string `json:"domain_status"`
	CreatedDate     string `json:"created_date"`
	ExpiryDate      string `json:"expiry_date"`
	DaysUntilExpiry int64  `json:"days_until_expiry"`
}

// Record 域名记录信息
type Record struct {
	CloudProvider string `json:"cloud_provider"`
	CloudName     string `json:"cloud_name"`
	DomainName    string `json:"domain_name"`
	RecordID      string `json:"record_id"`
	RecordType    string `json:"record_type"`
	RecordName    string `json:"record_name"`
	RecordValue   string `json:"record_value"`
	RecordTTL     string `json:"record_ttl"`
	RecordWeight  string `json:"record_weight"`
	RecordStatus  string `json:"record_status"`
	RecordRemark  string `json:"record_remark"`
	UpdateTime    string `json:"update_time"`
	FullRecord    string `json:"full_record"` // 完整记录 = Name + Value
}

type GetRecordCertReq struct {
	CloudProvider string `json:"cloud_provider"`
	CloudName     string `json:"cloud_name"`
	DomainName    string `json:"domain_name"`
	FullRecord    string `json:"full_record"`
	RecordID      string `json:"record_id"`
}

// RecordCert 域名证书信息
type RecordCert struct {
	CloudProvider             string `json:"cloud_provider"`
	CloudName                 string `json:"cloud_name"`
	DomainName                string `json:"domain_name"`                 // 域名
	FullRecord                string `json:"full_record"`                 // 完整记录 = Name + Value
	RecordID                  string `json:"record_id"`                   // 记录ID
	SubjectCommonName         string `json:"subject_common_name"`         // 颁发对象的公用名
	SubjectOrganization       string `json:"subject_organization"`        // 颁发对象的组织
	SubjectOrganizationalUnit string `json:"subject_organizational_unit"` // 颁发对象的组织单位
	IssuerCommonName          string `json:"issuer_common_name"`          // 颁发者的公用名
	IssuerOrganization        string `json:"issuer_organization"`         // 颁发者的组织
	IssuerOrganizationalUnit  string `json:"issuer_organizational_unit"`  // 颁发者的组织单位
	CreatedDate               string `json:"created_date"`                // 创建日期
	ExpiryDate                string `json:"expiry_date"`                 // 过期日期
	DaysUntilExpiry           int    `json:"days_until_expiry"`           // 距离到期日期还有多少天
	CertMatched               bool   `json:"cert_matched"`                // 证书是否匹配
	ErrorMsg                  string `json:"error_msg"`
}

// DNSProvider 接口定义
type DNSProvider interface {
	ListDomains() ([]Domain, error)
	ListRecords() ([]Record, error)
}

// DNSProviderFactory 用于注册和创建 DNSProvider 实例
type DNSProviderFactory struct {
	dnsProviders map[string]func(account map[string]string) DNSProvider
}

func NewDNSProviderFactory() *DNSProviderFactory {
	return &DNSProviderFactory{dnsProviders: make(map[string]func(account map[string]string) DNSProvider)}
}

// Register 注册 DNSProvider 实现
func (f *DNSProviderFactory) Register(cloudProvider string, factoryFunc func(account map[string]string) DNSProvider) {
	f.dnsProviders[strings.ToLower(cloudProvider)] = factoryFunc
}

// Create 创建 DNSProvider 实例
func (f *DNSProviderFactory) Create(cloudProvider string, account map[string]string) (DNSProvider, error) {
	if factory, exists := f.dnsProviders[strings.ToLower(cloudProvider)]; exists {
		return factory(account), nil
	}
	return nil, fmt.Errorf("unsupported cloud provider: %s", cloudProvider)
}

// 统一记录状态的值
func oneStatus(status string) string {
	// tencent 的记录状态是 ENABLE 和 DISABLE
	if status == "ENABLE" || status == "ACTIVE" {
		return "enable"
	}
	if status == "DISABLE" {
		return "disable"
	}
	return status
}
