package utils

import (
	"context"
	"fmt"
	"time"

	"bugyou-backend/internal/config"

	"go.mongodb.org/mongo-driver/bson"
)

func GenerateTicketID(ctx context.Context, issueType string) (string, error) {
	year := time.Now().Year()
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)
	prefix := "BUG"
	if issueType == "New Requirement" {
		prefix = "NR"
	}

	count, err := config.Collection("issues").CountDocuments(ctx, bson.M{
		"createdAt": bson.M{"$gte": start, "$lt": end},
		"ticketId":  bson.M{"$regex": fmt.Sprintf("^%s-%d-", prefix, year)},
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%d-%03d", prefix, year, count+1), nil
}
