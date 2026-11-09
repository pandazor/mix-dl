package model

func ValidLanguage(s string) bool {
	switch s {
	case "", "fr", "en", "es", "zh-CN", "ja":
		return true
	}
	return false
}
