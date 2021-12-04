package redis_test

import (
	"log"

	"github.com/go-broadcast/broadcast"
	"github.com/go-broadcast/redis"
)

func Example() {
	dispatcher, err := redis.New()
	if err != nil {
		log.Fatal(err)
	}

	broadcaster, cancel, err := broadcast.New(
		broadcast.WithDispatcher(dispatcher),
	)
	if err != nil {
		log.Fatal(err)
	}

	cancel()
	<-broadcaster.Done()
}
