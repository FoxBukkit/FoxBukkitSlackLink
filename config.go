package main

import (
	"encoding/json"
	"errors"
	"os"

	zmq "github.com/pebbe/zmq4"
)

func ParseConfig(filename string) (*Config, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	config := new(Config)
	if err = decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

var errUnknownZeroMQMode = errors.New("unknown ZeroMQ mode")

func ApplyZeroMQConfigs(socket *zmq.Socket, configs []*ZeroMQConfig) (err error) {
	for _, config := range configs {
		switch config.Mode {
		case "connect":
			err = socket.Connect(config.URI)
		case "bind":
			err = socket.Bind(config.URI)
		default:
			err = errUnknownZeroMQMode
		}

		if err != nil {
			return
		}
	}

	return
}

type ZeroMQConfig struct {
	Mode string `json:"mode"`
	URI  string `json:"uri"`
}

type Config struct {
	ZeroMQ struct {
		ServerToBroker []*ZeroMQConfig `json:"serverToBroker"`
		BrokerToServer []*ZeroMQConfig `json:"brokerToServer"`
	} `json:"zeromq"`

	Slack struct {
		Token string `json:"token"`
	} `json:"slack"`
}
