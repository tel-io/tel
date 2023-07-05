package ztrace

type config struct {
	TrackLogFields  bool
	TrackLogMessage bool
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithTrackLogFields(b bool) Option {
	return optionFunc(func(c *config) {
		c.TrackLogFields = b
	})
}

func WithTrackLogMessage(b bool) Option {
	return optionFunc(func(c *config) {
		c.TrackLogMessage = b
	})
}
