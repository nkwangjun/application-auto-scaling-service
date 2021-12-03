package utils

import "strings"

// IsInSlice 判断字符串是否在字符数组中，不区分大小写与首位空格
func IsInStrSlice(sliceStr []string, targetStr string) bool {
	for _, str := range sliceStr {
		if strings.EqualFold(str, targetStr) {
			return true
		}
	}
	return false
}
