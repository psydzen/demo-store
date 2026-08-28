package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// formInt64 reads a required numeric form field.
func formInt64(r *http.Request, name string) (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(r.PostFormValue(name)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse form value %s: %w", name, err)
	}
	return v, nil
}

// formIntDefault reads a numeric form field, falling back when it is empty or
// unparsable.
func formIntDefault(r *http.Request, name string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(r.PostFormValue(name)))
	if err != nil {
		return fallback
	}
	return v
}

// formText reads a trimmed text field.
func formText(r *http.Request, name string) string {
	return strings.TrimSpace(r.PostFormValue(name))
}

// formBool reports whether a checkbox was ticked.
func formBool(r *http.Request, name string) bool {
	switch strings.TrimSpace(r.PostFormValue(name)) {
	case "on", "true", "1", "yes":
		return true
	default:
		return false
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }
