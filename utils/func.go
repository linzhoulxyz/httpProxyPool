package utils

import (
	"strconv"
	"time"
	"unicode"
)

const (
	ISO_TIME_YYYYMMDDHHIISS = "2006-01-02 15:04:05"
	ISO_TIME_YYYYMMDD       = "2006-01-02"
)

// Str2Int 字符串转整型
func Str2Int(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}

	return i
}

// Str2Float64 字符串转浮点数
func Str2Float64(str string) float64 {
	i, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}

	return i
}

// IsHan 检查是否是中文
func IsHan(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

// 字符串转时间
func Str2Time(str string, format string) time.Time {
	if format == "" {
		format = ISO_TIME_YYYYMMDDHHIISS
	}
	t, e := time.ParseInLocation(format, str, time.Local)
	if e != nil {
		return time.Now()
	}

	return t
}
