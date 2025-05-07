package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func main() {
	// Secret key - should match what's configured in your API Gateway
	// In production, this would be a secure environment variable
	secretKey := []byte("your-secret-key-change-in-production")

	// Create claims with user information
	claims := jwt.MapClaims{
		"user_id": "test-user-123",
		"role":    "user",
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		"iat":     time.Now().Unix(),                     // Token issued at
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated JWT Token:")
	fmt.Println(tokenString)
	fmt.Println("\nUse this token in your requests:")
	fmt.Printf("curl -X POST http://localhost:8080/api/v1/streams \\\n")
	fmt.Printf("  -H \"Content-Type: application/json\" \\\n")
	fmt.Printf("  -H \"Authorization: Bearer %s\" \\\n", tokenString)
	fmt.Printf("  -d '{\n")
	fmt.Printf("    \"title\": \"Test Stream\",\n")
	fmt.Printf("    \"description\": \"This is a test stream\",\n")
	fmt.Printf("    \"visibility\": \"public\",\n")
	fmt.Printf("    \"tags\": [\"test\", \"dev\"]\n")
	fmt.Printf("  }'\n")
}
