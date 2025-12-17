package nacos

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/Iori372552686/GoOne/lib/contrib/config"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Option is nacos config option.
type Option func(o *options)

type options struct {
	ctx          context.Context
	group        string
	dataIDs      []string
	defaultFmt   string
	ignoreEmpty  bool
}

// WithContext sets context for nacos operations.
func WithContext(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

// WithGroup sets nacos group (default: DEFAULT_GROUP).
func WithGroup(group string) Option {
	return func(o *options) { o.group = group }
}

// WithDataIDs sets the config dataIds to load/watch (required).
func WithDataIDs(ids ...string) Option {
	return func(o *options) { o.dataIDs = append(o.dataIDs, ids...) }
}

// WithDefaultFormat sets the default format when a dataId has no extension (default: json).
func WithDefaultFormat(fmt string) Option {
	return func(o *options) { o.defaultFmt = fmt }
}

// WithIgnoreEmpty makes Load() skip empty config values (default: false).
func WithIgnoreEmpty(ignore bool) Option {
	return func(o *options) { o.ignoreEmpty = ignore }
}

type source struct {
	client  config_client.IConfigClient
	options *options
}

// New creates a nacos config Source.
func New(client config_client.IConfigClient, opts ...Option) (config.Source, error) {
	if client == nil {
		return nil, errors.New("nacos client is nil")
	}
	op := &options{
		ctx:        context.Background(),
		group:      "DEFAULT_GROUP",
		dataIDs:    nil,
		defaultFmt: "json",
	}
	for _, o := range opts {
		o(op)
	}
	op.group = strings.TrimSpace(op.group)
	if op.group == "" {
		op.group = "DEFAULT_GROUP"
	}
	op.defaultFmt = strings.TrimSpace(op.defaultFmt)
	if op.defaultFmt == "" {
		op.defaultFmt = "json"
	}
	// sanitize dataIds
	clean := make([]string, 0, len(op.dataIDs))
	seen := map[string]struct{}{}
	for _, id := range op.dataIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	op.dataIDs = clean
	if len(op.dataIDs) == 0 {
		return nil, errors.New("dataIds invalid (empty)")
	}
	return &source{client: client, options: op}, nil
}

func (s *source) Load() ([]*config.KeyValue, error) {
	kvs := make([]*config.KeyValue, 0, len(s.options.dataIDs))
	for _, dataID := range s.options.dataIDs {
		content, err := s.client.GetConfig(vo.ConfigParam{
			DataId: dataID,
			Group:  s.options.group,
		})
		if err != nil {
			return nil, err
		}
		if s.options.ignoreEmpty && strings.TrimSpace(content) == "" {
			continue
		}
		f := strings.TrimPrefix(filepath.Ext(dataID), ".")
		if f == "" {
			f = s.options.defaultFmt
		}
		kvs = append(kvs, &config.KeyValue{
			Key:    dataID,
			Value:  []byte(content),
			Format: f,
		})
	}
	return kvs, nil
}

func (s *source) Watch() (config.Watcher, error) {
	return newWatcher(s)
}


