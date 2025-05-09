package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWT generates a new JWT token with the given parameters.
func GenerateJWT(userID string, secret string, expiryDuration time.Duration, issuer string) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryDuration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAndValidateJWT parses a JWT token string, validates its signature and standard claims.
// It returns the RegisteredClaims if the token is valid, or an error otherwise.
func ParseAndValidateJWT(tokenString string, secretKey string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid // Or a more specific error like fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err // This will include errors like token expired, signature invalid, etc.
	}

	if !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid // Or a more generic error like errors.New("token is invalid")
	}

	return claims, nil
}
