package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventType categorizes what kind of usage event occurred.
type EventType string

const (
	EventLimitHit         EventType = "limit_hit"
	EventApproachingLimit EventType = "approaching_limit"
)

// ResourceType identifies which resource the event relates to.
type ResourceType string

const (
	ResourceAgent   ResourceType = "agent"
	ResourceMonitor ResourceType = "monitor"
)

// UsageEvent records when a user hits or approaches a plan limit.
type UsageEvent struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	EventType    EventType
	ResourceType ResourceType
	CurrentCount int
	MaxAllowed   int
	Plan         Plan
	CreatedAt    time.Time
}

// NewUsageEvent creates a new UsageEvent with generated ID and timestamp.
func NewUsageEvent(userID uuid.UUID, eventType EventType, resourceType ResourceType, currentCount, maxAllowed int, plan Plan) *UsageEvent {
	return &UsageEvent{
		ID:           uuid.New(),
		UserID:       userID,
		EventType:    eventType,
		ResourceType: resourceType,
		CurrentCount: currentCount,
		MaxAllowed:   maxAllowed,
		Plan:         plan,
		CreatedAt:    time.Now(),
	}
}
