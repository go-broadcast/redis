package redis

import (
	"context"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/go-broadcast/broadcast"
	"github.com/go-redis/redis/v8"
	"github.com/rs/xid"
)

// Option type defines a redis dispatcher option.
type Option func(d *redisDispatcher) error

// WithPrefix sets the prefix of the redis channel name.
// The channel name has the form <prefix>messages-<broadcaster ID>
// Default prefix is "broadcaster-"
func WithPrefix(prefix string) Option {
	return func(d *redisDispatcher) error {
		d.prefix = prefix
		return nil
	}
}

// WithDB sets the redis DB. Default is 0.
func WithDB(db int) Option {
	return func(d *redisDispatcher) error {
		d.redisOptions.DB = db
		return nil
	}
}

// WithAddr sets the redis instance address. Default is "localhost:6379".
func WithAddr(addr string) Option {
	return func(d *redisDispatcher) error {
		d.redisOptions.Addr = addr
		return nil
	}
}

// WithPassword sets the redis instance password. Default is "".
func WithPassword(password string) Option {
	return func(d *redisDispatcher) error {
		d.redisOptions.Password = password
		return nil
	}
}

// WithContext sets the parent Context for publish and subscribe operations.
func WithContext(ctx context.Context) Option {
	return func(d *redisDispatcher) error {
		d.context = ctx
		return nil
	}
}

// WithPublishTimeout sets the timeout when publishing messages to redis.
// Default is 3 seconds.
func WithPublishTimeout(timeout time.Duration) Option {
	return func(d *redisDispatcher) error {
		d.publishTimeout = timeout
		return nil
	}
}

// WithLogger sets the logger. By default logs are discarded.
func WithLogger(logger *log.Logger) Option {
	return func(d *redisDispatcher) error {
		d.logger = logger
		return nil
	}
}

// New creates a redis dispatcher.
func New(options ...Option) (broadcast.Dispatcher, error) {
	redisOptions := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	logger := log.New(ioutil.Discard, "go-broadcast", log.Ldate|log.Ltime)

	dispatcher := &redisDispatcher{
		id:             xid.New().String(),
		prefix:         "broadcaster-",
		redisOptions:   redisOptions,
		context:        context.Background(),
		logger:         logger,
		publishTimeout: time.Second * 3,
	}

	for _, option := range options {
		err := option(dispatcher)

		if err != nil {
			return nil, err
		}
	}

	dispatcher.client = redis.NewClient(dispatcher.redisOptions)
	dispatcher.sub = dispatcher.client.PSubscribe(dispatcher.context, dispatcher.channelPattern())

	go func() {
		dispatcher.logger.Println("start listening for redis messages")
		dispatcher.listenForMessages()
		dispatcher.logger.Println("stopped listening for redis messages")
	}()

	return dispatcher, nil
}

type redisDispatcher struct {
	id             string
	prefix         string
	logger         *log.Logger
	context        context.Context
	redisOptions   *redis.Options
	client         *redis.Client
	sub            *redis.PubSub
	publishTimeout time.Duration
	callback       func(data interface{}, toAll bool, room string, except ...string)
}

// Dispatch implementation of broadcast.Dispatcher.
func (d *redisDispatcher) Dispatch(data interface{}, toAll bool, room string, except ...string) {
	message := message{
		ToAll:  toAll,
		Room:   room,
		Except: except,
		Data:   data,
	}

	ctx, cancel := context.WithTimeout(d.context, d.publishTimeout)
	defer cancel()
	cmd := d.client.Publish(ctx, d.channelName(), &message)

	if cmd.Err() != nil {
		d.logger.Printf("error publishing message: %v", cmd.Err())
		return
	}
}

// Received implementation of broadcast.Dispatcher.
func (d *redisDispatcher) Received(callback func(data interface{}, toAll bool, room string, except ...string)) {
	d.callback = callback
}

func (d *redisDispatcher) channelName() string {
	return d.prefix + "messages-" + d.id
}

func (d *redisDispatcher) channelPattern() string {
	return d.prefix + "messages-*"
}

func (d *redisDispatcher) listenForMessages() {
	for {
		select {
		case redisMessage := <-d.sub.Channel():
			d.process(redisMessage)
		case <-d.context.Done():
			d.sub.Close()
			return
		}
	}
}

func (d *redisDispatcher) process(m *redis.Message) {
	if d.callback == nil {
		d.logger.Println("dispatcher callback not set")
		return
	}

	if strings.EqualFold(m.Channel, d.channelName()) {
		return
	}

	var payload message
	err := payload.UnmarshalBinary([]byte(m.Payload))

	if err != nil {
		d.logger.Printf("unable to unmarshal redis message %v", err)
		return
	}

	d.callback(payload.Data, payload.ToAll, payload.Room, payload.Except...)
}
