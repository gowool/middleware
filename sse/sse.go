package sse

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/gowool/wool"
	"github.com/gowool/wool/render"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
	"io"
	"time"
)

const (
	ClientKey      = "sse_client"
	EventConnected = "connected"
	EventClosing   = "closing"
)

type Config struct {
	Version    string        `mapstructure:"version"`
	Namespace  string        `mapstructure:"namespace"`
	ClientIdle time.Duration `mapstructure:"client_idle"`
}

func (cfg *Config) Init() {
	if cfg.Version == "" {
		cfg.Version = "(untracked)"
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "sse"
	}
}

type Client struct {
	ID        string
	Idle      time.Duration
	EventChan chan render.SSEvent
	Done      chan struct{}
}

type client struct {
	Client
	start time.Time
}

type message struct {
	clientID string
	event    render.SSEvent
}

type Event struct {
	cfg *Config
	log *slog.Logger

	done        chan struct{}
	notifier    chan message
	subscribe   chan client
	unsubscribe chan string
	clients     map[string]client

	clientsCount   *prometheus.GaugeVec
	clientDuration *prometheus.HistogramVec
	eventsCount    *prometheus.CounterVec
}

func New(cfg *Config, log *slog.Logger) *Event {
	cfg.Init()

	labels := []string{"version", "client"}

	e := &Event{
		cfg:         cfg,
		log:         log,
		done:        make(chan struct{}, 1),
		notifier:    make(chan message),
		subscribe:   make(chan client),
		unsubscribe: make(chan string),
		clients:     make(map[string]client),
		clientsCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Name:      "http_sse_clients_count",
				Help:      "HTTP SSE number of clients.",
			}, labels,
		),
		clientDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Name:      "http_sse_connection_duration_seconds",
				Help:      "HTTP SSE connection duration in seconds.",
			}, labels,
		),
		eventsCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "http_sse_events_count_total",
				Help:      "HTTP SSE total number of events.",
			}, labels,
		),
	}

	prometheus.MustRegister(e.clientsCount, e.clientDuration, e.eventsCount)

	go e.listen()

	return e
}

func (e *Event) Handler(c wool.Ctx) error {
	cl, ok := c.Get(ClientKey).(Client)
	if !ok {
		return errors.New("SSE client not found")
	}

	if err := c.SSEvent(EventConnected, cl.ID); err != nil {
		return err
	}
	c.Res().Flush()

	if cl.Idle == 0 {
		return c.Stream(func(w io.Writer) error {
			select {
			case <-cl.Done:
				if err := c.SSEvent(EventClosing, cl.ID); err != nil {
					return err
				}
				return wool.ErrStreamClosed
			case event := <-cl.EventChan:
				return c.Render(-1, event)
			}
		})
	}

	cancelCtx, cancelRequest := context.WithCancel(c.Req().Context())
	defer cancelRequest()

	c.SetReq(c.Req().WithContext(cancelCtx))

	// start an idle timer to keep track of inactive/forgotten connections
	idleTimer := time.NewTimer(cl.Idle)
	defer idleTimer.Stop()

	return c.Stream(func(w io.Writer) error {
		select {
		case <-idleTimer.C:
			cancelRequest()
		case <-cl.Done:
			if err := c.SSEvent(EventClosing, cl.ID); err != nil {
				return err
			}
			return wool.ErrStreamClosed
		case event := <-cl.EventChan:
			idleTimer.Stop()
			idleTimer.Reset(cl.Idle)

			return c.Render(-1, event)
		}
		return nil
	})
}

func (e *Event) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		clientID := c.Req().PathParamID()
		if clientID == "" {
			clientID = uuid.NewString()
		}

		cl := Client{
			ID:        clientID,
			Idle:      e.cfg.ClientIdle,
			EventChan: make(chan render.SSEvent),
			Done:      make(chan struct{}, 1),
		}

		e.Subscribe(cl)

		defer func() {
			go e.Unsubscribe(clientID)
			close(cl.EventChan)
			close(cl.Done)
		}()

		c.Set(ClientKey, cl)

		return next(c)
	}
}

func (e *Event) Broadcast(event render.SSEvent) {
	e.notifier <- message{clientID: "", event: event}
}

func (e *Event) Notify(clientID string, event render.SSEvent) {
	e.notifier <- message{clientID, event}
}

func (e *Event) Subscribe(cl Client) {
	e.subscribe <- client{Client: cl, start: time.Now()}
}

func (e *Event) Unsubscribe(clientID string) {
	e.unsubscribe <- clientID
}

func (e *Event) Close() error {
	e.done <- struct{}{}
	return nil
}

func (e *Event) listen() {
	for {
		select {
		case <-e.done:
			clients := e.clients
			for _, cl := range clients {
				cl.Done <- struct{}{}
			}
			return
		case msg := <-e.notifier:
			if msg.clientID == "" {
				e.broadcast(msg.event)
			} else {
				e.notify(msg.clientID, msg.event)
			}
		case cl := <-e.subscribe:
			e.sub(cl)
		case clientID := <-e.unsubscribe:
			e.unsub(clientID)
		}
	}
}

func (e *Event) broadcast(event render.SSEvent) {
	for id, _ := range e.clients {
		e.notify(id, event)
	}
}

func (e *Event) notify(clientID string, event render.SSEvent) {
	if cl, ok := e.clients[clientID]; ok {
		cl.EventChan <- event
		e.metricEvent(clientID, event)
	}
}

func (e *Event) sub(cl client) {
	e.unsub(cl.ID)
	e.clients[cl.ID] = cl
	e.metricSubscribe(cl)
}

func (e *Event) unsub(clientID string) {
	if cl, ok := e.clients[clientID]; ok {
		delete(e.clients, clientID)
		e.metricUnsubscribe(cl)
	} else {
		e.log.Debug("unsubscribe client not found", "client", clientID)
	}
}

func (e *Event) metricSubscribe(cl client) {
	e.clientsCount.WithLabelValues(e.cfg.Version, cl.ID).Inc()
	e.log.Info("subscribe client", "client", cl.ID, "start", cl.start)
}

func (e *Event) metricUnsubscribe(cl client) {
	duration := time.Since(cl.start).Seconds()

	e.clientDuration.WithLabelValues(e.cfg.Version, cl.ID).Observe(duration)
	e.clientsCount.WithLabelValues(e.cfg.Version, cl.ID).Dec()
	e.log.Info("unsubscribe client", "client", cl.ID, "duration_seconds", duration)
}

func (e *Event) metricEvent(clientId string, event render.SSEvent) {
	e.eventsCount.WithLabelValues(e.cfg.Version, clientId).Inc()
	e.log.Info("notify client", "client", clientId)
	e.log.Debug("notify client", "client", clientId, "event", event)
}
