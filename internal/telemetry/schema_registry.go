package telemetry

import (
	"context"
	"strings"

	"github.com/twmb/franz-go/pkg/sr"
)

type srClient struct {
	client *sr.Client
}

// NewSchemaRegistry talks HTTP to a Confluent-compatible registry.
func NewSchemaRegistry(url string) (SchemaRegistry, error) {
	cl, err := sr.NewClient(sr.URLs(url))
	if err != nil {
		return nil, err
	}
	return &srClient{client: cl}, nil
}

func (c *srClient) Register(ctx context.Context, subject, schema string) (int, error) {
	ss, err := c.client.CreateSchema(ctx, subject, sr.Schema{
		Schema: schema,
		Type:   sr.TypeProtobuf,
	})
	if err != nil {
		return 0, err
	}
	c.client.SetCompatibility(ctx, sr.SetCompatibility{Level: sr.CompatBackward}, subject)
	return ss.ID, nil
}

func (l *Logger) UseSchemaRegistry(url, topic string) error {
	if l == nil {
		return nil
	}
	url = strings.TrimSpace(url)
	topic = strings.TrimSpace(topic)
	if url == "" || topic == "" {
		return nil
	}
	reg, err := NewSchemaRegistry(url)
	if err != nil {
		return err
	}
	l.Event.serde = newConfluentSerde(reg, topic, l.Event.logf())
	return nil
}
