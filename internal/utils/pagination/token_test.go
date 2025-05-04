package pagination

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeToken(t *testing.T) {
	// Test case 1: Standard date/time values
	journalDate := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2023, 5, 15, 14, 30, 45, 123456789, time.UTC)

	// Encode the token
	token := EncodeToken(journalDate, createdAt)
	assert.NotEmpty(t, token, "Token should not be empty")

	// Decode the token and verify
	decodedJournalDate, decodedCreatedAt, err := DecodeToken(token)
	assert.NoError(t, err, "Decoding should not return an error")
	assert.Equal(t, journalDate, decodedJournalDate, "Journal date should match after decode")
	assert.Equal(t, createdAt, decodedCreatedAt, "Created at time should match after decode")

	// Test case 2: Zero time values
	zeroTime := time.Time{}
	zeroToken := EncodeToken(zeroTime, zeroTime)
	decodedZeroDate, decodedZeroTime, err := DecodeToken(zeroToken)
	assert.NoError(t, err, "Decoding zero time should not return an error")
	assert.Equal(t, zeroTime, decodedZeroDate, "Zero date should match after decode")
	assert.Equal(t, zeroTime, decodedZeroTime, "Zero time should match after decode")

	// Test case 3: Current time values
	now := time.Now().UTC()
	nowToken := EncodeToken(now, now)
	decodedNowDate, decodedNowTime, err := DecodeToken(nowToken)
	assert.NoError(t, err, "Decoding current time should not return an error")

	// Due to potential nanosecond precision issues, use Equal instead of direct comparison
	assert.True(t, now.Equal(decodedNowDate), "Current date should match after decode")
	assert.True(t, now.Equal(decodedNowTime), "Current time should match after decode")
}

func TestDecodeTokenError(t *testing.T) {
	// Test invalid base64
	_, _, err := DecodeToken("this is not base64!")
	assert.Error(t, err, "Should return an error for invalid base64")
	assert.Contains(t, err.Error(), "base64 decode", "Error should mention base64 decoding")

	// Test invalid format (missing separator)
	invalidToken := "MjAyMy0wNS0xNVQwMDowMDowMFo=" // Base64 encoded date without separator
	_, _, err = DecodeToken(invalidToken)
	assert.Error(t, err, "Should return an error for invalid token format")
	assert.Contains(t, err.Error(), "split", "Error should mention splitting issue")

	// Test invalid date format
	invalidDateToken := "bm90YWRhdGV8MjAyMy0wNS0xNVQxNDozMDo0NS4xMjM0NTY3ODla" // Base64 encoded "notadate|2023-05-15T14:30:45.123456789Z"
	_, _, err = DecodeToken(invalidDateToken)
	assert.Error(t, err, "Should return an error for invalid date format")
	assert.Contains(t, err.Error(), "journal date parse", "Error should mention date parsing issue")
}

func TestEncodeDateBasedToken(t *testing.T) {
	// Test with a known date
	testDate := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)
	token := EncodeDateBasedToken(testDate)

	decodedDate, err := DecodeDateBasedToken(token)
	assert.NoError(t, err, "Decoding should not return an error")
	assert.Equal(t, testDate, decodedDate, "Date should match after decode")

	// Test with current time
	now := time.Now().UTC()
	nowToken := EncodeDateBasedToken(now)

	decodedNow, err := DecodeDateBasedToken(nowToken)
	assert.NoError(t, err, "Decoding should not return an error")
	assert.True(t, now.Equal(decodedNow), "Date should match after decode")
}

func TestEncodeMultiFieldToken(t *testing.T) {
	// Test with simple fields
	fields := []string{"field1", "field2", "field3"}
	token := EncodeMultiFieldToken(fields...)

	decodedFields, err := DecodeMultiFieldToken(token)
	assert.NoError(t, err, "Decoding should not return an error")
	assert.Equal(t, fields, decodedFields, "Fields should match after decode")

	// Test with empty fields
	emptyToken := EncodeMultiFieldToken()
	decodedEmpty, err := DecodeMultiFieldToken(emptyToken)
	assert.NoError(t, err, "Decoding should not return an error")
	// When splitting an empty string with strings.Split, we get a slice with one empty string
	assert.Equal(t, []string{""}, decodedEmpty, "Should decode to slice with one empty string")

	// Test with special characters
	specialFields := []string{"field|with|pipes", "field with spaces", "field\nwith\nnewlines"}
	specialToken := EncodeMultiFieldToken(specialFields...)

	decodedSpecial, err := DecodeMultiFieldToken(specialToken)
	assert.NoError(t, err, "Decoding should not return an error")
	assert.Len(t, decodedSpecial, 5, "Should split on all pipe characters")

	// Test fields with timestamps
	timestampStr := time.Now().UTC().Format(time.RFC3339Nano)
	timeToken := EncodeMultiFieldToken("account123", timestampStr)

	decodedTime, err := DecodeMultiFieldToken(timeToken)
	assert.NoError(t, err, "Decoding should not return an error")
	assert.Equal(t, 2, len(decodedTime), "Should have decoded 2 fields")
	assert.Equal(t, "account123", decodedTime[0], "First field should match")
	assert.Equal(t, timestampStr, decodedTime[1], "Timestamp field should match")
}
