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
	"bugyou-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type createIssueRequest struct {
	Type             string `json:"type"`
	Product          string `json:"product"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Category         string `json:"category"`
	Priority         string `json:"priority"`
	AttachmentURL    string `json:"attachmentUrl"`
	BrowserDevice    string `json:"browserDevice"`
	Deadline         string `json:"deadline"`
	StepsToReproduce string `json:"stepsToReproduce"`
	ExpectedResult   string `json:"expectedResult"`
	ActualResult     string `json:"actualResult"`
}

func CreateIssue(c *gin.Context) {
	req, ok := parseCreateIssueRequest(c)
	if !ok {
		return
	}

	if message := validateIssueInput(req.Type, req.Product, req.Title, req.Description, req.Category, req.Priority); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": message})
		return
	}
	deadline, message := parseDeadline(req.Deadline)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": message})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	ticketID, err := utils.GenerateTicketID(ctx, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not generate ticket id"})
		return
	}

	now := time.Now().UTC()
	issue := models.Issue{
		ID:               primitive.NewObjectID(),
		TicketID:         ticketID,
		Type:             req.Type,
		Product:          req.Product,
		Title:            req.Title,
		Description:      req.Description,
		Category:         req.Category,
		Priority:         req.Priority,
		Status:           constants.StatusOpen,
		AttachmentURL:    req.AttachmentURL,
		BrowserDevice:    req.BrowserDevice,
		Deadline:         deadline,
		StepsToReproduce: req.StepsToReproduce,
		ExpectedResult:   req.ExpectedResult,
		ActualResult:     req.ActualResult,
		CreatedBy: models.CreatedBy{
			Name:  c.GetString("name"),
			Email: c.GetString("email"),
		},
		CreatedAt:         now,
		UpdatedAt:         now,
		ResolvedAt:        nil,
		DeveloperComments: []models.DeveloperComment{},
	}

	if _, err := config.Collection("issues").InsertOne(ctx, issue); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not create issue"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Your issue has been submitted successfully.",
		"ticketId": ticketID,
		"issue":    issue,
	})
}

func parseCreateIssueRequest(c *gin.Context) (createIssueRequest, bool) {
	var req createIssueRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err == nil && hasCreateIssueJSONFields(req) {
		return trimCreateIssueRequest(req), true
	}

	attachmentURL, ok := middleware.SaveOptionalAttachment(c)
	if !ok {
		return createIssueRequest{}, false
	}

	return createIssueRequest{
		Type:             strings.TrimSpace(c.PostForm("type")),
		Product:          strings.TrimSpace(c.PostForm("product")),
		Title:            strings.TrimSpace(c.PostForm("title")),
		Description:      strings.TrimSpace(c.PostForm("description")),
		Category:         strings.TrimSpace(c.PostForm("category")),
		Priority:         strings.TrimSpace(c.PostForm("priority")),
		AttachmentURL:    attachmentURL,
		BrowserDevice:    strings.TrimSpace(c.PostForm("browserDevice")),
		Deadline:         strings.TrimSpace(c.PostForm("deadline")),
		StepsToReproduce: strings.TrimSpace(c.PostForm("stepsToReproduce")),
		ExpectedResult:   strings.TrimSpace(c.PostForm("expectedResult")),
		ActualResult:     strings.TrimSpace(c.PostForm("actualResult")),
	}, true
}

func hasCreateIssueJSONFields(req createIssueRequest) bool {
	return req.Type != "" ||
		req.Product != "" ||
		req.Title != "" ||
		req.Description != "" ||
		req.Category != "" ||
		req.Priority != ""
}

func trimCreateIssueRequest(req createIssueRequest) createIssueRequest {
	req.Type = strings.TrimSpace(req.Type)
	req.Product = strings.TrimSpace(req.Product)
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	req.Priority = strings.TrimSpace(req.Priority)
	req.AttachmentURL = strings.TrimSpace(req.AttachmentURL)
	req.BrowserDevice = strings.TrimSpace(req.BrowserDevice)
	req.Deadline = strings.TrimSpace(req.Deadline)
	req.StepsToReproduce = strings.TrimSpace(req.StepsToReproduce)
	req.ExpectedResult = strings.TrimSpace(req.ExpectedResult)
	req.ActualResult = strings.TrimSpace(req.ActualResult)

	return req
}

func MyIssues(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cursor, err := config.Collection("issues").Find(ctx, bson.M{"createdBy.email": c.GetString("email")})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not fetch issues"})
		return
	}
	defer cursor.Close(ctx)

	var issues []models.Issue
	if err := cursor.All(ctx, &issues); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not read issues"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"issues": issues})
}

func TrackIssue(c *gin.Context) {
	ticketID := normalizeTicketID(c.Param("ticketId"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var issue models.Issue
	if err := config.Collection("issues").FindOne(ctx, bson.M{"ticketId": ticketID}).Decode(&issue); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "issue not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"issue": issue})
}

func validateIssueInput(typeValue string, product string, title string, description string, category string, priority string) string {
	if typeValue == "" || product == "" || title == "" || description == "" || category == "" || priority == "" {
		return "type, product, title, description, category and priority are required"
	}
	if len(title) < 5 {
		return "title must be at least 5 characters"
	}
	if len(description) < 10 {
		return "description must be at least 10 characters"
	}
	if !constants.IsAllowed(typeValue, constants.IssueTypes) {
		return "invalid issue type"
	}
	if !constants.IsAllowed(product, constants.Products) {
		return "invalid product"
	}
	if !constants.IsAllowed(category, constants.Categories) {
		return "invalid category"
	}
	if !constants.IsAllowed(priority, constants.Priorities) {
		return "invalid priority"
	}

	return ""
}

func parseDeadline(value string) (*time.Time, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ""
	}

	deadline, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, "deadline must use YYYY-MM-DD format"
	}

	deadline = time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.UTC)
	return &deadline, ""
}

func normalizeTicketID(value string) string {
	ticketID := strings.ToUpper(strings.TrimSpace(value))
	if strings.HasPrefix(ticketID, "BUG-") || strings.HasPrefix(ticketID, "NR-") {
		return ticketID
	}
	return ticketID
}
