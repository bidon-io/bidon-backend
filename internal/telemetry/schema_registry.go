package telemetry

import (
	"context"
	"strings"

	"github.com/twmb/franz-go/pkg/sr"
	"go.uber.org/zap"
)

type srClient struct {
	client *sr.Client
	log    *zap.Logger
}

// NewSchemaRegistry talks HTTP to a Confluent-compatible registry.
func NewSchemaRegistry(url string, log *zap.Logger) (SchemaRegistry, error) {
	cl, err := sr.NewClient(sr.URLs(url))
	if err != nil {
		return nil, err
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &srClient{client: cl, log: log}, nil
}

// Register returns the schema id even if setting BACKWARD compatibility fails:
// the id is valid, and failing here would fall back to unframed records.
func (c *srClient) Register(ctx context.Context, subject, schema string) (int, error) {
	ss, err := c.client.CreateSchema(ctx, subject, sr.Schema{
		Schema: schema,
		Type:   sr.TypeProtobuf,
	})
	if err != nil {
		return 0, err
	}
	for _, r := range c.client.SetCompatibility(ctx, sr.SetCompatibility{Level: sr.CompatBackward}, subject) {
		if r.Err != nil {
			c.log.Warn("schema registry set compatibility",
				zap.Error(r.Err),
				zap.String("subject", subject),
				zap.String("level", sr.CompatBackward.String()),
			)
		}
	}
	return ss.ID, nil
}

// UseSchemaRegistry frames telemetry-events values for the registry at url.
// Subjects register in the background; events emitted before a subject has
// an id are produced as raw protobuf.
func (l *Logger) UseSchemaRegistry(url, topic string) error {
	if l == nil {
		return nil
	}
	url = strings.TrimSpace(url)
	topic = strings.TrimSpace(topic)
	if url == "" || topic == "" {
		return nil
	}
	reg, err := NewSchemaRegistry(url, l.event.logf())
	if err != nil {
		return err
	}
	l.event.serde = newConfluentSerde(reg, topic, l.event.logf())
	go l.event.serde.registerAll(context.Background())
	return nil
}
