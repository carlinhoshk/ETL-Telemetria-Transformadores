package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"etl-telemetria-transformadores/internal/messaging"
)

// ErrTopicMismatch rejects payloads whose transformer_id does not match the
// topic the message arrived on.
var ErrTopicMismatch = errors.New("payload transformer_id does not match topic")

// Metrics are basic counters for accepted/rejected traffic (reported via
// periodic logs; Prometheus exposition comes in the observability phase).
type Metrics struct {
	Received    atomic.Uint64
	Accepted    atomic.Uint64
	Rejected    atomic.Uint64
	Duplicates  atomic.Uint64
	StoreErrors atomic.Uint64
}

// Snapshot returns a plain map for logging.
func (m *Metrics) Snapshot() map[string]uint64 {
	return map[string]uint64{
		"received":     m.Received.Load(),
		"accepted":     m.Accepted.Load(),
		"rejected":     m.Rejected.Load(),
		"duplicates":   m.Duplicates.Load(),
		"store_errors": m.StoreErrors.Load(),
	}
}

// Ingestor runs the pipeline MQTT → validation → normalization →
// persistence (bronze) with structured logs, metrics, idempotency and
// graceful shutdown.
type Ingestor struct {
	logger  *slog.Logger
	client  mqtt.Client
	val     *Validator
	store   Store
	metrics *Metrics
	seen    sync.Map // dedup keys (transformer_id@timestamp)
	source  string
}

// NewIngestor prepares the pipeline. Connect() must be called before Run().
func NewIngestor(logger *slog.Logger, val *Validator, store Store, source string) *Ingestor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Ingestor{logger: logger, val: val, store: store, metrics: &Metrics{}, source: source}
}

// Connect dials the broker. Clean session is disabled so QoS 1 messages
// redelivered after a reconnect are still handled (and deduplicated here).
func (i *Ingestor) Connect(broker, clientID string) error {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(false).
		SetConnectTimeout(5 * time.Second).
		SetAutoReconnect(true).
		SetOnConnectHandler(func(_ mqtt.Client) {
			i.logger.Info("mqtt connected", "broker", broker)
		})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("connect to %s: %w", broker, token.Error())
	}
	i.client = client
	return nil
}

// Run subscribes to transformer telemetry and blocks until ctx is done,
// then unsubscribes and disconnects gracefully.
func (i *Ingestor) Run(ctx context.Context) error {
	token := i.client.Subscribe(messaging.TelemetrySubscribeFilter, 1, i.handleMessage)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe %s: %w", messaging.TelemetrySubscribeFilter, token.Error())
	}
	i.logger.Info("subscribed", "filter", messaging.TelemetrySubscribeFilter)

	<-ctx.Done()
	i.logger.Info("shutting down")
	token = i.client.Unsubscribe(messaging.TelemetrySubscribeFilter)
	token.Wait()
	i.client.Disconnect(250)
	i.logger.Info("disconnected")
	return ctx.Err()
}

// MetricsSnapshot returns the current counters for logging/monitoring.
func (i *Ingestor) MetricsSnapshot() map[string]uint64 {
	return i.metrics.Snapshot()
}

// handleMessage processes a single MQTT message through the pipeline.
func (i *Ingestor) handleMessage(_ mqtt.Client, msg mqtt.Message) {
	i.metrics.Received.Add(1)

	topicID, topicOK := messaging.ParseTelemetryTopic(msg.Topic())
	m, err := i.val.Validate(msg.Payload())
	if err != nil {
		i.reject("validation", msg.Topic(), err)
		return
	}
	if !topicOK || topicID != m.TransformerID {
		i.reject("topic", msg.Topic(), ErrTopicMismatch)
		return
	}

	key := DedupKey(m)
	if _, loaded := i.seen.LoadOrStore(key, struct{}{}); loaded {
		i.metrics.Duplicates.Add(1)
		i.logger.Debug("duplicate skipped", "transformer_id", m.TransformerID, "key", key)
		return
	}

	m = Normalize(m)
	now := time.Now().UTC()

	if err := i.store.WriteRaw(context.Background(), RawRecord{
		ID:            key,
		TransformerID: m.TransformerID,
		SchemaVersion: m.SchemaVersion,
		Topic:         msg.Topic(),
		Source:        i.source,
		ReceivedAt:    now.Format(time.RFC3339),
		Payload:       json.RawMessage(msg.Payload()),
	}); err != nil {
		i.metrics.StoreErrors.Add(1)
		i.logger.Error("raw write failed", "transformer_id", m.TransformerID, "error", err)
		return
	}
	if err := i.store.WriteMeasurement(context.Background(), m); err != nil {
		i.metrics.StoreErrors.Add(1)
		i.logger.Error("measurement write failed", "transformer_id", m.TransformerID, "error", err)
		return
	}

	i.metrics.Accepted.Add(1)
	i.logger.Info("accepted",
		"transformer_id", m.TransformerID,
		"timestamp", m.Timestamp,
		"state", m.State,
		"topic", msg.Topic(),
	)
}

func (i *Ingestor) reject(reason, topic string, err error) {
	i.metrics.Rejected.Add(1)
	i.logger.Warn("message rejected",
		slog.String("reason", reason),
		slog.String("topic", topic),
		slog.String("error", err.Error()),
	)
}
