This package allows you to scale out [broadcast](https://github.com/go-broadcast/broadcast) using Redis.

## Installation

```bash
go get github.com/go-broadcast/redis
```

## Basic usage

```go
import (
	"log"

	"github.com/go-broadcast/broadcast"
	"github.com/go-broadcast/redis"
)

func main() {
	dispatcher, err := redis.New()
	if err != nil {
		log.Fatal(err)
	}
	
	broadcaster, err := broadcast.New(
		redis.WithDispatcher(dispatcher),
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

## Full example

[Scale out with Redis](https://github.com/go-broadcast/examples/tree/main/cmd/redis)
