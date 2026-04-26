package watermill

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

type PubSub interface {
	message.Publisher
	message.Subscriber
	Close() error
}

func NewInProcessBus() (PubSub, func(), error) {
	logger := watermill.NewStdLogger(false, false)
	ps := gochannel.NewGoChannel(gochannel.Config{}, logger)
	return ps, func() {
		_ = ps.Close()
	}, nil
}
