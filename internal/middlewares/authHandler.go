package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func VerifyJWT(publicKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		headerArr := strings.SplitN(authHeader, " ", 2)
		if len(headerArr) != 2 || headerArr[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
			return
		}

		tokenStr := headerArr[1]

		if publicKey == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Public key missing"})
			return
		}

		// Parse public key
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse public key"})
			return
		}

		// Parse and verify token
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return pubKey, nil
		})

		if err != nil || !token.Valid {
			fmt.Println("JWT Parse Error:", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// (Optional) Save claims into Gin Context
		claims, ok := token.Claims.(jwt.MapClaims)
		if ok {
			c.Set("user", claims)
		}

		userId, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in claims"})
			return
		}

		// (Optional) Save userId in your DB (e.g., creating or updating a user record)
		// Assuming you have a User model and a database connection
		// Example using GORM (replace with your DB logic):
		// db := c.MustGet("db").(*gorm.DB) // Assuming you have the DB connection set in Gin's context
		// user := User{UserID: userId}    // Create or fetch user by userId
		// db.Save(&user)

		// Save the userId in the Gin context for use in other handlers
		c.Set("userId", userId)

		c.Next()
	}
}

func DevAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", "test123")
		c.Next()
	}
}
