package pagination

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const timeFormat = time.RFC3339Nano // Use a precise time format

// EncodeToken creates a base64 encoded token from a journal date and creation time.
// This is used for consistent pagination across different repositories.
func EncodeToken(journalDate time.Time, createdAt time.Time) string {
	tokenStr := fmt.Sprintf("%s|%s", journalDate.Format(timeFormat), createdAt.Format(timeFormat))
	return base64.StdEncoding.EncodeToString([]byte(tokenStr))
}

// DecodeToken parses the base64 encoded token back into journal date and creation time.
func DecodeToken(token string) (time.Time, time.Time, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (base64 decode): %w", err)
	}
	tokenStr := string(decodedBytes)
	parts := strings.SplitN(tokenStr, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (split)")
	}

	journalDate, err := time.Parse(timeFormat, parts[0])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (journal date parse): %w", err)
	}

	createdAt, err := time.Parse(timeFormat, parts[1])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (created_at parse): %w", err)
	}

	return journalDate, createdAt, nil
}

// EncodeDateBasedToken creates a token for single date field pagination
func EncodeDateBasedToken(date time.Time) string {
	return base64.StdEncoding.EncodeToString([]byte(date.Format(timeFormat)))
}

// DecodeDateBasedToken decodes a token for single date field pagination
func DecodeDateBasedToken(token string) (time.Time, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid pagination token format (base64 decode): %w", err)
	}

	date, err := time.Parse(timeFormat, string(decodedBytes))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid pagination token format (date parse): %w", err)
	}

	return date, nil
}

// EncodeMultiFieldToken creates a token with any number of string fields
// This provides flexibility for different pagination strategies
func EncodeMultiFieldToken(fields ...string) string {
	tokenStr := strings.Join(fields, "|")
	return base64.StdEncoding.EncodeToString([]byte(tokenStr))
}

// DecodeMultiFieldToken decodes a token into its component fields
func DecodeMultiFieldToken(token string) ([]string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination token format (base64 decode): %w", err)
	}

	tokenStr := string(decodedBytes)
	parts := strings.Split(tokenStr, "|")
	return parts, nil
}
