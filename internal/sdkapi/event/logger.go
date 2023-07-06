package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
)

type Logger struct {
	Engine LoggerEngine
}

type LoggerEngine interface {
	Produce(ctx context.Context, topic Topic, message []byte, errorCb func(error)) error
}

func (l *Logger) Log(c echo.Context, event Event) error {
	payload := make(map[string]any)
	smashMap(payload, event.Payload)

	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %v", err)
	}

	err = l.Engine.Produce(c.Request().Context(), event.Topic, message, func(err error) {
		logError(c, fmt.Errorf("async produce message: %v", err))
	})
	if err != nil {
		return fmt.Errorf("produce message: %v", err)
	}

	return nil
}

func smashMap(dst, src map[string]any, nesting ...string) {
	prefix := strings.Join(nesting, "__")

	for key, value := range src {
		mapValue, ok := value.(map[string]any)
		if ok {
			n := slices.Clone(nesting)
			n = append(n, key)
			smashMap(dst, mapValue, n...)
		} else if prefix != "" {
			dst[fmt.Sprintf("%s__%s", prefix, key)] = value
		} else {
			dst[key] = value
		}
	}
}
