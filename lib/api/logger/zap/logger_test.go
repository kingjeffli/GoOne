/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package zap

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func reset() {
	setLogger(nil)
}

func TestInitLogger(t *testing.T) {
	config := Config{
		Level: "debug",
	}
	_, err := InitLogger(config)
	assert.NoError(t, err)
	reset()
}

func TestGetLogger(t *testing.T) {
	// not yet init get default log
	log := GetLogger()
	config := Config{
		Level: "debug",
	}
	_, _ = InitLogger(config)
	// after init logger
	log2 := GetLogger()
	assert.NotEqual(t, log, log2)

	// the secend init logger
	config.Level = "info"
	_, _ = InitLogger(config)
	log3 := GetLogger()
	assert.NotEqual(t, log2, log3)
	reset()
}

func TestSetLogger(t *testing.T) {
	// not yet init get default log
	log := GetLogger()
	log1 := &mockLogger{}
	setLogger(log1)

	// after set logger
	log2 := GetLogger()
	assert.NotEqual(t, log, log2)
	assert.Equal(t, log1, log2)

	config := Config{
		Level: "degug",
	}
	_, _ = InitLogger(config)
	// after init logger
	log3 := GetLogger()
	assert.NotEqual(t, log2, log3)
	reset()
}

func TestRaceLogger(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			setLogger(&mockLogger{})
		}()
		go func() {
			defer wg.Done()
			_ = GetLogger()
		}()
		go func() {
			defer wg.Done()
			config := Config{
				Level: "debug",
			}
			_, _ = InitLogger(config)
		}()
	}
	wg.Wait()
	reset()
}

func TestGetLogWriter_NoStdout(t *testing.T) {
	cfg := Config{LogDir: t.TempDir(), LogFileName: "a.log", LogStdout: false}
	w := cfg.getLogWriter()
	assert.NotNil(t, w)
}

func TestGetLogWriter_WithStdout_IsMultiWriter(t *testing.T) {
	cfg := Config{LogDir: t.TempDir(), LogFileName: "a.log", LogStdout: true}
	w := cfg.getLogWriter()
	assert.NotNil(t, w)

	// zapcore.NewMultiWriteSyncer returns a zapcore.WriteSyncer that is not the plain file syncer.
	// We can't introspect its children (type is unexported), but we can ensure it's a usable WriteSyncer.
	assert.Implements(t, (*zapcore.WriteSyncer)(nil), w)
}

func TestNewConsoleEncoderConfig(t *testing.T) {
	enc := newConsoleEncoderConfig()
	assert.Equal(t, "time", enc.TimeKey)
	assert.Equal(t, "level", enc.LevelKey)
	assert.Equal(t, "caller", enc.CallerKey)
	assert.Equal(t, "msg", enc.MessageKey)
}

type mockLogger struct {
}

func (m mockLogger) Info(_ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Warn(_ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Error(_ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Debug(_ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Fatal(_ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Sync() error {
	return nil
}

func (m mockLogger) Infof(_ string, _ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Warnf(_ string, _ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Errorf(_ string, _ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Debugf(_ string, _ ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Fatalf(_ string, _ ...interface{}) {
	panic("implement me")
}
