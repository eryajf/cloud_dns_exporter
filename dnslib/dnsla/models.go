package dnsla

import (
	"net/url"
	"strconv"
)

// DomainResponse 解析记录列表响应
type DomainListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Total   int      `json:"total"`
		Results []Domain `json:"results"`
	} `json:"data"`
}

// Domain 域名
type Domain struct {
	ID            string `json:"id"`            // 域名ID
	CreatedAt     int64  `json:"createdAt"`     // 域名添加时间 Unix 时间戳
	UpdatedAt     int64  `json:"updatedAt"`     // 域名最后修改时间 Unix 时间戳
	UserID        string `json:"userId"`        // 用户ID
	UserAccount   string `json:"userAccount"`   // 用户账号
	AssetID       string `json:"assetId"`       // 域名当前生效套餐的资产ID
	GroupID       string `json:"groupId"`       // 分组ID,空为默认分组
	GroupName     string `json:"groupName"`     // 分组名称,空为默认分组
	Domain        string `json:"domain"`        // Punycode 编码后的域名
	DisplayDomain string `json:"displayDomain"` // Punycode 编码前的域名
	State         int    `json:"state"`         // 域名状态 1 正常 | 2 暂停 其他状态咨询客服
	NsState       int    `json:"nsState"`       // 域名NS状态 0 未知 | 1 匹配 | 2 未匹配 | 3 未加入
	NsCheckedAt   int64  `json:"nsCheckedAt"`   // 域名NS检查时间
	ProductCode   string `json:"productCode"`   // 域名套餐代码
	ProductName   string `json:"productName"`   // 域名套餐名称
	ExpiredAt     int64  `json:"expiredAt"`     // 过期时间 Unix 时间戳,免费域名过期时间 2100-01-01
	QuoteDomainID string `json:"quoteDomainId"` // 引用域名ID
	QuoteDomain   string `json:"quoteDomain"`   // 引用域名
	Suffix        string `json:"suffix"`        // Punycode 编码后的顶级域名
	DisplaySuffix string `json:"displaySuffix"` // Punycode 编码前的顶级域名
}

// RecordListResponse 解析记录列表响应
type RecordListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Total   int      `json:"total"`
		Results []Record `json:"results"`
	} `json:"data"`
}

// Record 解析记录
type Record struct {
	ID          string `json:"id"`          // 记录ID
	CreatedAt   int64  `json:"createdAt"`   // 记录添加时间 Unix 时间戳
	UpdatedAt   int64  `json:"updatedAt"`   // 记录最后修改时间 Unix 时间戳
	DomainID    string `json:"domainId"`    // 域名ID
	GroupID     string `json:"groupId"`     // 分组ID,空为默认分组
	GroupName   string `json:"groupName"`   // 分组名称,空为默认分组
	Host        string `json:"host"`        // Punycode 编码后的主机头
	DisplayHost string `json:"displayHost"` // Punycode 编码前的主机头
	Type        int    `json:"type"`        // 记录类型
	LineID      string `json:"lineId"`      // 线路id，参考线路文档
	LineCode    string `json:"lineCode"`    // 线路code，参考线路文档
	LineName    string `json:"lineName"`    // 线路名称，参考线路文档
	Data        string `json:"data"`        // Punycode 编码后的记录值
	DisplayData string `json:"displayData"` // Punycode 编码前的记录值
	TTL         int    `json:"ttl"`         // TTL
	Weight      int    `json:"weight"`      // 权重
	Preference  int    `json:"preference"`  // MX优先级
	Dominant    bool   `json:"domaint"`     // 是否显性URL转发
	System      bool   `json:"system"`      // 是否系统解析记录
	Disable     bool   `json:"disable"`     // 是否暂停
}

// DomainListOption 获取域名列表的参数
type DomainListOption func(url.Values)

func DLWithGroupID(groupID string) DomainListOption {
	return func(v url.Values) {
		if groupID != "" {
			v.Set("groupId", groupID)
		}
	}
}

func DLWithState(state int) DomainListOption {
	return func(v url.Values) {
		v.Set("state", strconv.Itoa(state))
	}
}

func DLWithProductCode(productCode string) DomainListOption {
	return func(v url.Values) {
		if productCode != "" {
			v.Set("productCode", productCode)
		}
	}
}

func DLWithQuoteDomainID(quoteDomainID string) DomainListOption {
	return func(v url.Values) {
		if quoteDomainID != "" {
			v.Set("quoteDomainId", quoteDomainID)
		}
	}
}

func DLWithExpiredAtRange(begin, end int64) DomainListOption {
	return func(v url.Values) {
		if begin != 0 {
			v.Set("expiredAtBegin", strconv.FormatInt(begin, 10))
		}
		if end != 0 {
			v.Set("expiredAtEnd", strconv.FormatInt(end, 10))
		}
	}
}

// RecordListOption 获取解析记录列表的参数
type RecordListOption func(url.Values)

func RLWithRecordType(recordType int) RecordListOption {
	return func(v url.Values) {
		v.Set("type", strconv.Itoa(recordType))
	}
}

func RLWithGroupID(groupID string) RecordListOption {
	return func(v url.Values) {
		if groupID != "" {
			v.Set("groupId", groupID)
		}
	}
}

func RLWithLineID(lineID string) RecordListOption {
	return func(v url.Values) {
		if lineID != "" {
			v.Set("lineId", lineID)
		}
	}
}

func RLWithHost(host string) RecordListOption {
	return func(v url.Values) {
		if host != "" {
			v.Set("host", host)
		}
	}
}

func RLWithData(data string) RecordListOption {
	return func(v url.Values) {
		if data != "" {
			v.Set("data", data)
		}
	}
}

func RLWithDisable(disable bool) RecordListOption {
	return func(v url.Values) {
		v.Set("disable", strconv.FormatBool(disable))
	}
}

func RLWithSystem(system bool) RecordListOption {
	return func(v url.Values) {
		v.Set("system", strconv.FormatBool(system))
	}
}

func RLWithDominant(dominant bool) RecordListOption {
	return func(v url.Values) {
		v.Set("dominant", strconv.FormatBool(dominant))
	}
}
