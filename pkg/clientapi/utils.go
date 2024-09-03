package clientapi

import (
	"strings"
)

const (
	missingData = "404"
	notFound    = "NOT_FOUND"
)

func response404(err string) bool {
	return strings.Contains(err, missingData) || strings.Contains(err, notFound)
}
