package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type config struct {
	port         int
	kafkaBrokers []string
}

func loadConfig() (config, error) {
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil || port < 1 || port > 65535 {
		return config{}, errors.New("PORT must be an integer between 1 and 65535")
	}

	brokerList := strings.TrimSpace(os.Getenv("KAFKA_BROKERS"))
	if brokerList == "" {
		return config{}, errors.New("KAFKA_BROKERS must contain at least one host:port address")
	}

	brokers := strings.Split(brokerList, ",")
	for i, broker := range brokers {
		broker = strings.TrimSpace(broker)
		host, brokerPort, err := net.SplitHostPort(broker)
		number, portErr := strconv.Atoi(brokerPort)
		if err != nil || host == "" || strings.ContainsAny(host, "/\\?#@") ||
			strings.ContainsFunc(host, unicode.IsSpace) ||
			portErr != nil || number < 1 || number > 65535 {
			return config{}, fmt.Errorf("KAFKA_BROKERS entry %d must be a host:port address with a port between 1 and 65535", i+1)
		}
		brokers[i] = broker
	}

	return config{port: port, kafkaBrokers: brokers}, nil
}
