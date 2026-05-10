package domain

import "time"

type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

type EventBus interface {
	Publish(event DomainEvent) error
}

type EventHandler interface {
	EventName() string
	Handle(event DomainEvent) error
}

type UserDeactivated struct {
	userID     UserID
	occurredAt time.Time
}

func NewUserDeactivated(userID UserID) UserDeactivated {
	return UserDeactivated{
		userID:     userID,
		occurredAt: time.Now(),
	}
}

func (e UserDeactivated) UserID() UserID        { return e.userID }
func (e UserDeactivated) EventName() string     { return "user.deactivated" }
func (e UserDeactivated) OccurredAt() time.Time { return e.occurredAt }
