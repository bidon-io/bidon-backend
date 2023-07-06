package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaConfig struct {
	ClientOpts []kgo.Opt
	Topics     map[event.Topic]string
}

func Kafka() (conf KafkaConfig, err error) {
	seeds := strings.Split(os.Getenv("KAFKA_BROKERS_LIST"), ", ")
	clientID := os.Getenv("KAFKA_CLIENT_ID")
	linger, err := strconv.Atoi(os.Getenv("KAFKA_DELIVERY_INTERVAL"))
	if err != nil {
		return conf, fmt.Errorf("invalid KAFKA_DELIVERY_INTERVAL: %v", err)
	}

	conf.ClientOpts = []kgo.Opt{
		kgo.AllowAutoTopicCreation(),
		kgo.SeedBrokers(seeds...),
		kgo.ClientID(clientID),
		kgo.ProducerLinger(time.Second * time.Duration(linger)),
	}

	configTopic := os.Getenv("KAFKA_CONFIG_TOPIC")
	if configTopic == "" {
		return conf, fmt.Errorf("empty KAFKA_CONFIG_TOPIC: %v", err)
	}

	conf.Topics = map[event.Topic]string{
		event.ConfigTopic: configTopic,
	}

	return
}
