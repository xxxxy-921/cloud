package channel

import "fmt"

// Payload represents the message content to send through a channel.
type Payload struct {
	To      []string
	Subject string
	Body    string
	HTML    string
}

// Driver defines the interface for message channel implementations.
type Driver interface {
	Send(config map[string]any, payload Payload) error
	Test(config map[string]any) error
}

var drivers = map[string]Driver{
	"email": &EmailDriver{},
}

// GetDriver returns the driver for the given channel type.
func GetDriver(channelType string) (Driver, error) {
	d, ok := drivers[channelType]
	if !ok {
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
	return d, nil
}
