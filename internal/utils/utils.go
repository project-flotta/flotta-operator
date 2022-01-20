package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HasFinalizer checks whether specific finalizer is set on a CR
func HasFinalizer(cr *metav1.ObjectMeta, name string) bool {
	for _, f := range cr.GetFinalizers() {
		if f == name {
			return true
		}
	}
	return false
}

// NormalizeLabel normalizes label string according to k8s format
func NormalizeLabel(name string) (string, error) {
	// convert name to lowercase
	name = strings.ToLower(name)

	// slice string based on first and last alphanumeric character
	firstLegal := strings.IndexFunc(name, func(c rune) bool { return unicode.IsLower(c) || unicode.IsDigit(c) })
	lastLegal := strings.LastIndexFunc(name, func(c rune) bool { return unicode.IsLower(c) || unicode.IsDigit(c) })

	if firstLegal < 0 {
		return "", fmt.Errorf("The name doesn't contain a legal alphanumeric character")
	}

	name = name[firstLegal : lastLegal+1]
	reg := regexp.MustCompile("[^a-z0-9-_.]+")
	name = reg.ReplaceAllString(name, "")

	return name, nil
}
