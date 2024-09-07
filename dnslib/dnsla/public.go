package dnsla

type PageOption struct {
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
}

// NewPageOption 创建一个分页参数
func NewPageOption(pageIndex, pageSize int) PageOption {
	if !(pageSize > 0 && pageSize <= 1000) || pageIndex < 0 || pageSize <= 0 {
		return PageOption{
			PageIndex: 1,
			PageSize:  1000,
		}
	}
	return PageOption{
		PageIndex: pageIndex,
		PageSize:  pageSize,
	}
}
