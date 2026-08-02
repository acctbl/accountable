package apierror

import "strings"

func ValidMessageKey(key string) bool {
	if key == "" || strings.ContainsAny(key, " \t\n\r") {
		return false
	}
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for i, r := range part {
			ok := (r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') ||
				(i > 0 && r == '_')
			if !ok {
				return false
			}
		}
		if part[0] >= '0' && part[0] <= '9' {
			return false
		}
	}
	return true
}
