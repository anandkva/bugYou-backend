package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreatedBy struct {
	Name  string `bson:"name" json:"name"`
	Email string `bson:"email" json:"email"`
}

type AssignedUser struct {
	ID    string `bson:"id" json:"id"`
	Name  string `bson:"name" json:"name"`
	Email string `bson:"email" json:"email"`
}

type DeveloperComment struct {
	OldStatus string    `bson:"oldStatus" json:"oldStatus"`
	NewStatus string    `bson:"newStatus" json:"newStatus"`
	Comment   string    `bson:"comment" json:"comment"`
	UpdatedBy string    `bson:"updatedBy" json:"updatedBy"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type Issue struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TicketID          string             `bson:"ticketId" json:"ticketId"`
	Type              string             `bson:"type" json:"type"`
	Product           string             `bson:"product" json:"product"`
	Title             string             `bson:"title" json:"title"`
	Description       string             `bson:"description" json:"description"`
	Category          string             `bson:"category" json:"category"`
	Priority          string             `bson:"priority" json:"priority"`
	Status            string             `bson:"status" json:"status"`
	AttachmentURL     string             `bson:"attachmentUrl,omitempty" json:"attachmentUrl,omitempty"`
	BrowserDevice     string             `bson:"browserDevice,omitempty" json:"browserDevice,omitempty"`
	Deadline          *time.Time         `bson:"deadline,omitempty" json:"deadline,omitempty"`
	StepsToReproduce  string             `bson:"stepsToReproduce,omitempty" json:"stepsToReproduce,omitempty"`
	ExpectedResult    string             `bson:"expectedResult,omitempty" json:"expectedResult,omitempty"`
	ActualResult      string             `bson:"actualResult,omitempty" json:"actualResult,omitempty"`
	AssignedTo        *AssignedUser      `bson:"assignedTo,omitempty" json:"assignedTo,omitempty"`
	CreatedBy         CreatedBy          `bson:"createdBy" json:"createdBy"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
	ResolvedAt        *time.Time         `bson:"resolvedAt" json:"resolvedAt"`
	DeveloperComments []DeveloperComment `bson:"developerComments" json:"developerComments"`
}
