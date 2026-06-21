package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bugyou-backend/internal/config"
	"bugyou-backend/internal/constants"
	"bugyou-backend/internal/models"
	"bugyou-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Dashboard(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	filter := dashboardFilter(c)
	openCount, err := config.Collection("issues").CountDocuments(ctx, withStatus(filter, constants.StatusOpen))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not count open issues"})
		return
	}
	onHoldCount, err := config.Collection("issues").CountDocuments(ctx, withStatus(filter, constants.StatusOnHold))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not count on hold issues"})
		return
	}
	resolvedCount, err := config.Collection("issues").CountDocuments(ctx, withStatus(filter, constants.StatusResolved))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not count resolved issues"})
		return
	}
	rejectedCount, err := config.Collection("issues").CountDocuments(ctx, withStatus(filter, constants.StatusRejected))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not count rejected issues"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filter": filter,
		"summary": gin.H{
			"totalOpenBugs":     openCount,
			"totalOnHoldBugs":   onHoldCount,
			"totalResolvedBugs": resolvedCount,
			"totalRejectedBugs": rejectedCount,
		},
	})
}

func AllIssues(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	filter := issueListFilter(c)
	findOptions := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := config.Collection("issues").Find(ctx, filter, findOptions)
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

func UpdateStatus(c *gin.Context) {
	var req struct {
		Status  string `json:"status" binding:"required"`
		Comment string `json:"comment" binding:"required,min=5"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	req.Status = strings.TrimSpace(req.Status)
	req.Comment = strings.TrimSpace(req.Comment)
	if !constants.IsAllowed(req.Status, constants.Statuses) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid status"})
		return
	}
	if len(req.Comment) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "comment must be at least 5 characters"})
		return
	}

	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid issue id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var issue models.Issue
	if err := config.Collection("issues").FindOne(ctx, bson.M{"_id": id}).Decode(&issue); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "issue not found"})
		return
	}
	if c.GetString("role") == constants.RoleDeveloper && (issue.AssignedTo == nil || issue.AssignedTo.Email != c.GetString("email")) {
		c.JSON(http.StatusForbidden, gin.H{"message": "developers can update only assigned tickets"})
		return
	}

	now := time.Now().UTC()
	var resolvedAt *time.Time
	if req.Status == constants.StatusResolved {
		resolvedAt = &now
	}

	comment := models.DeveloperComment{
		OldStatus: issue.Status,
		NewStatus: req.Status,
		Comment:   req.Comment,
		UpdatedBy: c.GetString("email"),
		UpdatedAt: now,
	}

	update := bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"updatedAt":  now,
			"resolvedAt": resolvedAt,
		},
		"$push": bson.M{"developerComments": comment},
	}

	if _, err := config.Collection("issues").UpdateByID(ctx, id, update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated successfully", "comment": comment})
}

func Reminders(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cutoff := time.Now().UTC().AddDate(0, 0, -10)
	filter := bson.M{
		"status":    bson.M{"$in": []string{constants.StatusOpen, constants.StatusOnHold}},
		"createdAt": bson.M{"$lte": cutoff},
	}

	cursor, err := config.Collection("issues").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not fetch reminders"})
		return
	}
	defer cursor.Close(ctx)

	now := time.Now().UTC()
	var issues []models.Issue
	if err := cursor.All(ctx, &issues); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not read reminders"})
		return
	}

	reminders := make([]gin.H, 0, len(issues))
	for _, issue := range issues {
		reminders = append(reminders, gin.H{
			"issue":       issue,
			"pendingDays": utils.PendingDays(issue, now),
		})
	}

	c.JSON(http.StatusOK, gin.H{"reminders": reminders})
}

func IssueTrend(c *gin.Context) {
	days := 30
	if c.Query("range") == "last-week" {
		days = 7
	}

	now := time.Now().UTC()
	start := startOfDay(now).AddDate(0, 0, -(days - 1))
	match := bson.M{"createdAt": bson.M{"$gte": start}}

	product := strings.TrimSpace(c.Query("product"))
	if product != "" && product != "All Products" {
		match["product"] = product
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	pipeline := mongoPipeline(
		bson.M{"$match": match},
		bson.M{"$group": bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"}},
				"type": "$type",
			},
			"count": bson.M{"$sum": 1},
		}},
	)

	cursor, err := config.Collection("issues").Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not build issue trend"})
		return
	}
	defer cursor.Close(ctx)

	type aggregateRow struct {
		ID struct {
			Date string `bson:"date"`
			Type string `bson:"type"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}

	var rows []aggregateRow
	if err := cursor.All(ctx, &rows); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not read issue trend"})
		return
	}

	seriesByDate := map[string]gin.H{}
	for day := 0; day < days; day++ {
		date := start.AddDate(0, 0, day).Format("2006-01-02")
		seriesByDate[date] = gin.H{"date": date, "total": 0, "bugs": 0, "requirements": 0}
	}

	for _, row := range rows {
		point, ok := seriesByDate[row.ID.Date]
		if !ok {
			continue
		}

		point["total"] = point["total"].(int) + row.Count
		if row.ID.Type == "Bug" {
			point["bugs"] = point["bugs"].(int) + row.Count
		}
		if row.ID.Type == "New Requirement" {
			point["requirements"] = point["requirements"].(int) + row.Count
		}
	}

	series := make([]gin.H, 0, days)
	maxCount := 0
	totalCreated := 0
	for day := 0; day < days; day++ {
		date := start.AddDate(0, 0, day).Format("2006-01-02")
		point := seriesByDate[date]
		total := point["total"].(int)
		if total > maxCount {
			maxCount = total
		}
		totalCreated += total
		series = append(series, point)
	}

	c.JSON(http.StatusOK, gin.H{
		"range":        c.DefaultQuery("range", "last-month"),
		"days":         days,
		"product":      product,
		"totalCreated": totalCreated,
		"maxCount":     maxCount,
		"series":       series,
	})
}

// MyTasks — returns issues assigned to the logged-in developer.
func MyTasks(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	email := c.GetString("email")
	filter := bson.M{"assignedTo.email": email}
	findOptions := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := config.Collection("issues").Find(ctx, filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not fetch tasks"})
		return
	}
	defer cursor.Close(ctx)

	var issues []models.Issue
	if err := cursor.All(ctx, &issues); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not read tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"issues": issues})
}

// SelfAssign — developer self-assigns an unassigned ticket.
func SelfAssign(c *gin.Context) {
	issueID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid issue id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	assigned := models.AssignedUser{
		ID:    c.GetString("userIDHex"),
		Name:  c.GetString("name"),
		Email: c.GetString("email"),
	}

	update := bson.M{
		"$set": bson.M{
			"assignedTo": assigned,
			"updatedAt":  time.Now().UTC(),
		},
	}

	result, err := config.Collection("issues").UpdateByID(ctx, issueID, update)
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "issue not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned to you successfully"})
}

func mongoPipeline(stages ...bson.M) []bson.M {
	return stages
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dashboardFilter(c *gin.Context) bson.M {
	filter := bson.M{}
	switch c.DefaultQuery("filter", "all") {
	case "last-week":
		filter["createdAt"] = bson.M{"$gte": time.Now().UTC().AddDate(0, 0, -7)}
	case "last-month":
		filter["createdAt"] = bson.M{"$gte": time.Now().UTC().AddDate(0, 0, -30)}
	}

	product := strings.TrimSpace(c.Query("product"))
	if product != "" && product != "All Products" {
		filter["product"] = product
	}

	return filter
}

func withStatus(filter bson.M, status string) bson.M {
	copy := bson.M{}
	for key, value := range filter {
		copy[key] = value
	}
	copy["status"] = status
	return copy
}

func issueListFilter(c *gin.Context) bson.M {
	filter := bson.M{}
	addExactFilter(c, filter, "product", "All Products")
	addExactFilter(c, filter, "status", "All Status")
	addExactFilter(c, filter, "priority", "All Priority")
	addExactFilter(c, filter, "type", "All Types")
	addExactFilter(c, filter, "category", "All Categories")

	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	dateFilter := bson.M{}
	if startDate != "" {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			dateFilter["$gte"] = parsed
		}
	}
	if endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
			dateFilter["$lte"] = parsed.Add(24*time.Hour - time.Nanosecond)
		}
	}
	if len(dateFilter) > 0 {
		filter["createdAt"] = dateFilter
	}

	return filter
}

func addExactFilter(c *gin.Context, filter bson.M, key string, allValue string) {
	value := strings.TrimSpace(c.Query(key))
	if value != "" && value != allValue {
		filter[key] = value
	}
}
