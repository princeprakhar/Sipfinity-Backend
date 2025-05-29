package utils

import (
	"regexp"
	"strings"
)

func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func IsValidPassword(password string) bool {
	return len(password) >= 8
}

func IsValidRole(role string) bool {
	validRoles := []string{"admin", "customer"}
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

func SanitizeString(input string) string {
	return strings.TrimSpace(input)
}

func IsValidRating(rating int) bool {
	return rating >= 1 && rating <= 5
}