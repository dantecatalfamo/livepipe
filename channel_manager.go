package main

import (
	"errors"
	"fmt"
	"os"
)

type ChannelManager struct {
	Channels []*Channel
}

func NewChannelManager(filter string) (*ChannelManager, error) {
	stdoutChannel := NewChannel("stdout", nil)
	stdoutChannel.Output = os.Stdout
	stdoutChannel.OutputFilename = "stdout"
	stdoutChannel.ID = "stdout"

	if err := stdoutChannel.SetFilter(filter); err != nil {
		return nil, fmt.Errorf("setting initial filter: %w", err)
	}

	inputChannel := NewChannel("stdin", nil)
	inputChannel.ID = "stdin"

	return &ChannelManager{
		Channels: []*Channel{inputChannel, stdoutChannel},
	}, nil
}

func (manager *ChannelManager) IngestLine(line string) error {
	for _, channel := range manager.Channels {
		channel.IngestLine(line)
	}

	return nil
}

func (manager *ChannelManager) ChannelByID(id string) (*Channel, error) {
	for _, channel := range manager.Channels {
		if id == channel.ID {
			return channel, nil
		}
	}

	return nil, errors.New("could not find channel")
}

func (manager *ChannelManager) AddChannel(channel *Channel) {
	manager.Channels = append(manager.Channels, channel)
}
