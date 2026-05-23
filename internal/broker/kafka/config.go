package kafka

import "errors"

var (
	ErrMissingBrokers = errors.New("kafka brokers are required")
	ErrMissingTopic   = errors.New("kafka topic is required")
	ErrMissingGroupID = errors.New("kafka group id is required")
)

type ConsumerConfig struct {
	Brokers  []string
	Topic    string
	GroupID  string
	ClientID string
}

func (c ConsumerConfig) Validate() error {
	if len(c.Brokers) == 0 {
		return ErrMissingBrokers
	}

	if c.Topic == "" {
		return ErrMissingTopic
	}

	if c.GroupID == "" {
		return ErrMissingGroupID
	}

	return nil
}
