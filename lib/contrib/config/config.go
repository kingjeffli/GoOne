package config

// KeyValue is a configuration item.
// Format is a hint for decoding (e.g. "json", "yaml").
type KeyValue struct {
	Key    string
	Value  []byte
	Format string
}

// Source is a configuration source (config center, file, etc.).
// It matches the classic Kratos config.Source shape but is implemented locally
// to avoid external ecosystem coupling.
type Source interface {
	Load() ([]*KeyValue, error)
	Watch() (Watcher, error)
}

// Watcher watches a Source for changes.
type Watcher interface {
	Next() ([]*KeyValue, error)
	Stop() error
}

// Client is a config Source with an optional Close hook.
// Many sources are stateless and can use a no-op Close.
type Client interface {
	Source
	Close() error
}

type client struct {
	Source
	closeFn func() error
}

func (c *client) Close() error {
	if c.closeFn != nil {
		return c.closeFn()
	}
	return nil
}

// Wrap converts a Source into a Client (optionally with a Close hook).
func Wrap(src Source, closeFn func() error) Client {
	return &client{Source: src, closeFn: closeFn}
}

