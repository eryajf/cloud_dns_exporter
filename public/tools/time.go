package tools

import "time"

// GetReadTimeMs 将毫秒的时间戳转换为时间
func GetReadTimeMs(s int64) string {
	return time.Unix(0, s*int64(time.Millisecond)).Format("2006-01-02 15:04:05")
}
