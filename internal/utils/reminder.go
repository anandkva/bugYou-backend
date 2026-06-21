package utils

import (
	"time"

	"bugyou-backend/internal/constants"
	"bugyou-backend/internal/models"
)

func PendingDays(issue models.Issue, now time.Time) int {
	return int(now.Sub(issue.CreatedAt).Hours() / 24)
}

func IsReminderIssue(issue models.Issue, now time.Time) bool {
	if issue.Status != constants.StatusOpen && issue.Status != constants.StatusOnHold {
		return false
	}

	return PendingDays(issue, now) >= 10
}
