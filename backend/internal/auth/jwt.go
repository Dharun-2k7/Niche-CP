package auth

import (
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret []byte
	jwtOnce   sync.Once
)

func getJWTSecret() []byte {
	jwtOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			// In production, missing JWT_SECRET should be a hard crash
			if os.Getenv("ENV") == "production" || os.Getenv("ENV") == "prod" {
				log.Fatal("FATAL: JWT_SECRET environment variable is missing in production!")
			}
			jwtSecret = []byte("super_secret_fallback_key_for_dev") // Fallback for local development
		} else {
			jwtSecret = []byte(secret)
		}
	})
	return jwtSecret
}

// GenerateToken creates a JWT for a given user ID
func GenerateToken(userID int, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Expires in 72 hours
	})

	return token.SignedString(getJWTSecret())
}

// ValidateToken parses and validates a JWT, returning the user_id if valid
func ValidateToken(tokenString string) (int, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getJWTSecret(), nil
	})

	if err != nil {
		return 0, "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDF, okID := claims["user_id"].(float64) // JSON numbers are float64
		email, okEmail := claims["email"].(string)

		if !okID || !okEmail {
			return 0, "", errors.New("invalid or missing token claims")
		}

		return int(userIDF), email, nil
	}

	return 0, "", errors.New("invalid token")
}
