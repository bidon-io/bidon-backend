package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/sr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	telemetryproto "github.com/bidon-io/bidon-backend/schemas/proto"
)

const (
	schemaRegisterTimeout = 2 * time.Second
	registerRetryMin      = time.Second
	registerRetryMax      = 30 * time.Second
)

var confluentHeader sr.ConfluentHeader

// SchemaRegistry registers protobuf schemas and returns Confluent schema ids.
type SchemaRegistry interface {
	Register(ctx context.Context, subject, schema string) (id int, err error)
}

// confluentSerde frames values with schema ids registered off the request
// path by registerAll. frame only reads the cache, so a slow or down registry
// never delays an emit.
type confluentSerde struct {
	registry SchemaRegistry
	topic    string
	schema   string
	log      *zap.Logger
	retryMin time.Duration

	mu  sync.RWMutex
	ids map[string]int
}

func newConfluentSerde(registry SchemaRegistry, topic string, log *zap.Logger) *confluentSerde {
	if log == nil {
		log = zap.NewNop()
	}
	return &confluentSerde{
		registry: registry,
		topic:    topic,
		schema:   telemetryproto.EventsProto,
		log:      log,
		retryMin: registerRetryMin,
		ids:      make(map[string]int),
	}
}

func schemaSubject(topic string, msg proto.Message) string {
	return subjectFor(topic, msg.ProtoReflect().Descriptor())
}

func subjectFor(topic string, d protoreflect.MessageDescriptor) string {
	return topic + "-" + string(d.FullName())
}

func protoMessageIndex(msg proto.Message) []int {
	d := msg.ProtoReflect().Descriptor()
	var path []int
	for {
		path = append([]int{d.Index()}, path...)
		parent := d.Parent()
		md, ok := parent.(protoreflect.MessageDescriptor)
		if !ok {
			break
		}
		d = md
	}
	return path
}

// frame prefixes payload with the Confluent header, or returns it raw when
// the subject has no schema id yet.
func (s *confluentSerde) frame(msg proto.Message, payload []byte) []byte {
	if s == nil || s.registry == nil {
		return payload
	}

	subject := schemaSubject(s.topic, msg)
	id, ok := s.id(subject)
	if !ok {
		return payload
	}

	header, err := confluentHeader.AppendEncode(nil, id, protoMessageIndex(msg))
	if err != nil {
		s.log.Error("confluent frame", zap.Error(err), zap.String("subject", subject))
		return payload
	}
	return append(header, payload...)
}

func (s *confluentSerde) id(subject string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.ids[subject]
	return id, ok
}

// registerAll registers every catalog subject, retrying failures with
// exponential backoff until all have an id or ctx is done.
func (s *confluentSerde) registerAll(ctx context.Context) {
	pending := make([]string, 0, len(catalogTypes()))
	for _, d := range catalogTypes() {
		pending = append(pending, subjectFor(s.topic, d))
	}

	backoff := s.retryMin
	for {
		pending = s.registerPending(ctx, pending)
		if len(pending) == 0 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, registerRetryMax)
	}
}

func (s *confluentSerde) registerPending(ctx context.Context, subjects []string) []string {
	var failed []string
	for _, subject := range subjects {
		regCtx, cancel := context.WithTimeout(ctx, schemaRegisterTimeout)
		id, err := s.registry.Register(regCtx, subject, s.schema)
		cancel()
		if err != nil {
			s.log.Warn("schema registry register; producing raw protobuf until it succeeds",
				zap.Error(err),
				zap.String("subject", subject),
			)
			failed = append(failed, subject)
			continue
		}
		s.mu.Lock()
		s.ids[subject] = id
		s.mu.Unlock()
	}
	return failed
}

// stripConfluentPrefix removes the Confluent protobuf header when present.
// Raw proto (no magic byte) is returned unchanged.
func stripConfluentPrefix(b []byte) []byte {
	_, rest, err := confluentHeader.DecodeID(b)
	if err != nil {
		return b
	}
	_, payload, err := confluentHeader.DecodeIndex(rest, 8)
	if err != nil {
		return rest
	}
	return payload
}
