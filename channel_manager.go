package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
)

type ChannelManager struct {
	Channels []*Channel
}

func NewChannelManager(filter string) (*ChannelManager, error) {
	stdoutChannel := NewChannel("stdout", nil, nil, "")
	stdoutChannel.Output = os.Stdout
	stdoutChannel.OutputFilename = "stdout"
	stdoutChannel.ID = "stdout"

	if err := stdoutChannel.SetFilter(filter); err != nil {
		return nil, fmt.Errorf("setting initial filter: %w", err)
	}

	inputChannel := NewChannel("stdin", nil, nil, "")
	inputChannel.ID = "stdin"

	return &ChannelManager{
		Channels: []*Channel{inputChannel, stdoutChannel},
	}, nil
}

func (manager *ChannelManager) IngestString(str string) error {
	for _, channel := range manager.Channels {
		channel.IngestString(str)
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

func (manager *ChannelManager) AddChannel(channel *Channel) error {
	stdin, err := manager.ChannelByID("stdin")
	if err != nil {
		return fmt.Errorf("no stdin channel: %w", err)
	}

	for _, line := range stdin.History() {
		channel.IngestLine(line)
	}

	manager.Channels = append(manager.Channels, channel)

	return nil
}

func (manager *ChannelManager) RemoveChannel(id string) {
	manager.Channels = slices.DeleteFunc(manager.Channels, func(c *Channel) bool {
		if c.ID == id {
			for broadcast := range c.Broadcasts {
				c.RemoveBroadcast(broadcast)
				close(broadcast)
			}

			return true
		}

		return false
	})
}
