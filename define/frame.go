package define

import (
	"github.com/EmptyDea-Team/EmptyDea-core-client"
	"github.com/asaskevich/EventBus"
)

type Frame interface {
	Client() *client.Client
	EventBus() EventBus.Bus
	Run() error
	Close() error
}

type Task interface {
	Name() string
	Frame() Frame
	Start() error
	Pause() error
	Resume() error
	Close() error
}
