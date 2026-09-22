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

const schemaRegisterTimeout = 2 * time.Second

var confluentHeader sr.ConfluentHeader

// SchemaRegistry registers protobuf schemas and returns Confluent schema ids.
type SchemaRegistry interface {
	Register(ctx context.Context, subject, schema string) (id int, err error)
}

type confluentSerde struct {
	registry SchemaRegistry
	topic    string
	schema   string
	log      *zap.Logger

	mu  sync.Mutex
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
		ids:      make(map[string]int),
	}
}

func schemaSubject(topic string, msg proto.Message) string {
	return topic + "-" + messageTypeName(msg)
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

func (s *confluentSerde) frame(msg proto.Message, payload []byte) []byte {
	if s == nil || s.registry == nil {
		return payload
	}

	subject := schemaSubject(s.topic, msg)
	id, err := s.lookup(subject)
	if err != nil {
		s.log.Error("schema registry register",
			zap.Error(err),
			zap.String("subject", subject),
		)
		return payload
	}

	header, err := confluentHeader.AppendEncode(nil, id, protoMessageIndex(msg))
	if err != nil {
		s.log.Error("confluent frame", zap.Error(err), zap.String("subject", subject))
		return payload
	}
	return append(header, payload...)
}

func (s *confluentSerde) lookup(subject string) (int, error) {
	s.mu.Lock()
	if id, ok := s.ids[subject]; ok {
		s.mu.Unlock()
		return id, nil
	}
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), schemaRegisterTimeout)
	defer cancel()

	id, err := s.registry.Register(ctx, subject, s.schema)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	s.ids[subject] = id
	s.mu.Unlock()
	return id, nil
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
