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
