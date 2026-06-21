package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bugyou-backend/internal/config"
	"bugyou-backend/internal/constants"
	"bugyou-backend/internal/middleware"
	"bugyou-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	role := req.Role
	if role == "" {
		role = constants.RoleUser
	}
	if role != constants.RoleUser && role != constants.RoleDeveloper && role != constants.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"message": "role must be user, developer, or admin"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	email := strings.ToLower(strings.TrimSpace(req.Email))
	existing, err := config.Collection("users").CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not check existing user"})
		return
	}
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "email is already registered"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not secure password"})
		return
	}

	now := time.Now().UTC()
	user := models.User{
		ID:        primitive.NewObjectID(),
		Name:      strings.TrimSpace(req.Name),
		Email:     email,
		Password:  string(passwordHash),
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := config.Collection("users").InsertOne(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not register user"})
		return
	}

	token, err := createToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not create token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err := config.Collection("users").FindOne(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(req.Email))}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid email or password"})
		return
	}

	token, err := createToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

func Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"id":    c.GetString("userIDHex"),
		"name":  c.GetString("name"),
		"email": c.GetString("email"),
		"role":  c.GetString("role"),
	})
}

func createToken(user models.User) (string, error) {
	claims := middleware.Claims{
		UserID: user.ID.Hex(),
		Email:  user.Email,
		Name:   user.Name,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Values.JWTSecret))
}
