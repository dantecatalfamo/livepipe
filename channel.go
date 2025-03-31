package main

import (
	"container/ring"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"time"
)

type Channel struct {
	Name           string
	ID             string
	Filter         *regexp.Regexp
	LineHistory    *ring.Ring
	Output         io.WriteCloser
	OutputFilename string
	Broadcasts     map[chan Line]struct{}
}

type Line struct {
	Time  time.Time `json:"time"`
	Text  string    `json:"text,omitempty"`
	Event string    `json:"event,omitempty"`
}

func NewChannel(name string, filter *regexp.Regexp) *Channel {
	return &Channel{
		Name:        name,
		ID:          generateID(),
		Filter:      filter,
		LineHistory: ring.New(DefaultLineHistory),
		Broadcasts:  make(map[chan Line]struct{}),
	}
}

func (c *Channel) IngestLine(text string) error {
	if c.Filter != nil && !c.Filter.Match([]byte(text)) {
		return nil
	}
	line := Line{Text: text, Time: time.Now()}
	if err := c.AppendLine(line); err != nil {
		return err
	}

	return nil
}

func (c *Channel) AppendLine(line Line) error {
	// Maybe add a lock?
	c.LineHistory.Value = line
	c.LineHistory = c.LineHistory.Next()
	// Don't output a blank line when it's an event
	if c.Output != nil && line.Event == "" {
		if _, err := fmt.Fprintln(c.Output, line.Text); err != nil {
			return fmt.Errorf("channel %s: failed to write: %w", c.Name, err)
		}
	}
	for b := range c.Broadcasts {
		select {
		case b <- line:
		default:
		}
	}

	return nil
}

func (c *Channel) History() []Line {
	history := make([]Line, 0, c.LineHistory.Len())

	c.LineHistory.Do(func(value any) {
		line, ok := value.(Line)
		if !ok {
			return
		}
		history = append(history, line)
	})

	return history
}

func (c *Channel) SetFilter(filter string) error {
	regexp, err := regexp.Compile(filter)
	if err != nil {
		return fmt.Errorf("could not compile filter regex: %w", err)
	}

	oldFilter := ""
	if c.Filter != nil {
		oldFilter = c.Filter.String()
	}
	if oldFilter == regexp.String() {
		// No change
		return nil
	}

	c.Filter = regexp

	c.AppendLine(Line{
		Time:  time.Now(),
		Event: fmt.Sprintf("changed filter: %s -> %s", oldFilter, regexp.String()),
	})

	return nil
}

func (c *Channel) SetName(name string) {
	c.Name = name
}

func (c *Channel) AddBroadcast(b chan Line) {
	c.Broadcasts[b] = struct{}{}
}

func (c *Channel) RemoveBroadcast(b chan Line) {
	delete(c.Broadcasts, b)
}

func generateID() string {
	var bytes [16]byte
	rand.Read(bytes[:])

	return hex.EncodeToString(bytes[:])
}
