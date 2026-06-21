package controllers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"bugyou-backend/internal/config"
	"bugyou-backend/internal/constants"
	"bugyou-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// SeedAdmin creates the very first admin account.
// Protected by ADMIN_SECRET env var — pass it as the "secret" JSON field.
func SeedAdmin(c *gin.Context) {
	var req struct {
		Secret   string `json:"secret" binding:"required"`
		Name     string `json:"name" binding:"required,min=2"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret == "" || req.Secret != adminSecret {
		c.JSON(http.StatusForbidden, gin.H{"message": "invalid admin secret"})
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

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not secure password"})
		return
	}

	now := time.Now().UTC()
	user := models.User{
		ID:        primitive.NewObjectID(),
		Name:      strings.TrimSpace(req.Name),
		Email:     email,
		Password:  string(hash),
		Role:      constants.RoleAdmin,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := config.Collection("users").InsertOne(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not create admin"})
		return
	}

	token, err := createToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not create token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

// CreateDeveloper — admin creates a developer account.
func CreateDeveloper(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required,min=2"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
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

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not secure password"})
		return
	}

	now := time.Now().UTC()
	user := models.User{
		ID:        primitive.NewObjectID(),
		Name:      strings.TrimSpace(req.Name),
		Email:     email,
		Password:  string(hash),
		Role:      constants.RoleDeveloper,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := config.Collection("users").InsertOne(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not create developer"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "developer created successfully", "user": user})
}

// ListDevelopers — admin fetches all developer accounts.
func ListDevelopers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	cursor, err := config.Collection("users").Find(ctx, bson.M{"role": constants.RoleDeveloper})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not fetch developers"})
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not read developers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"developers": users})
}

// AssignDeveloper — admin assigns (or unassigns) a developer to a ticket.
func AssignDeveloper(c *gin.Context) {
	var req struct {
		DeveloperID string `json:"developerId"` // empty string = unassign
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	issueID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid issue id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var updateFields bson.M
	if req.DeveloperID == "" {
		// Unassign
		updateFields = bson.M{"$unset": bson.M{"assignedTo": ""}, "$set": bson.M{"updatedAt": time.Now().UTC()}}
	} else {
		devID, err := primitive.ObjectIDFromHex(req.DeveloperID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid developer id"})
			return
		}

		var dev models.User
		if err := config.Collection("users").FindOne(ctx, bson.M{"_id": devID, "role": constants.RoleDeveloper}).Decode(&dev); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "developer not found"})
			return
		}

		updateFields = bson.M{
			"$set": bson.M{
				"assignedTo": models.AssignedUser{
					ID:    dev.ID.Hex(),
					Name:  dev.Name,
					Email: dev.Email,
				},
				"updatedAt": time.Now().UTC(),
			},
		}
	}

	result, err := config.Collection("issues").UpdateByID(ctx, issueID, updateFields)
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "issue not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "developer assigned successfully"})
}

// DeleteIssue — admin hard-deletes a ticket.
func DeleteIssue(c *gin.Context) {
	issueID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid issue id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result, err := config.Collection("issues").DeleteOne(ctx, bson.M{"_id": issueID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not delete issue"})
		return
	}
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "issue not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "issue deleted successfully"})
}
