package nacos

import (
	"context"

	"github.com/Iori372552686/GoOne/lib/contrib/config"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type watcher struct {
	source    *source
	ch        chan struct{}
	closeChan chan struct{}
}

func newWatcher(s *source) (config.Watcher, error) {
	w := &watcher{
		source:    s,
		ch:        make(chan struct{}, 1),
		closeChan: make(chan struct{}),
	}

	for _, dataID := range s.options.dataIDs {
		// OnChange is called by nacos internal goroutine
		err := s.client.ListenConfig(vo.ConfigParam{
			DataId: dataID,
			Group:  s.options.group,
			OnChange: func(_, _, _, _ string) {
				select {
				case w.ch <- struct{}{}:
				default:
				}
			},
		})
		if err != nil {
			_ = w.Stop()
			return nil, err
		}
	}

	return w, nil
}

func (w *watcher) Next() ([]*config.KeyValue, error) {
	select {
	case <-w.ch:
		return w.source.Load()
	case <-w.closeChan:
		return nil, context.Canceled
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.closeChan:
		// already closed
		return nil
	default:
		close(w.closeChan)
	}
	// best-effort cancel listeners
	for _, dataID := range w.source.options.dataIDs {
		_ = w.source.client.CancelListenConfig(vo.ConfigParam{
			DataId: dataID,
			Group:  w.source.options.group,
		})
	}
	return nil
}


