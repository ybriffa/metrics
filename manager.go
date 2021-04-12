package metrics

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/ybriffa/metrics/driver"
)

var (
	ErrNotRegistered error = errors.New("not registered")
)

type manager struct {
	registers     map[string]*driver.Registry
	senders       []driver.Driver
	flushCh       []chan struct{}
	flushInterval time.Duration
	flushMutex    sync.Mutex

	wg sync.WaitGroup
	l  sync.RWMutex

	cancel context.CancelFunc
}

func (m *manager) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	// Create a goroutine to watch every senders
	for _, s := range m.senders {
		m.wg.Add(1)
		m.flushMutex.Lock()
		flushCh := make(chan struct{}, 1)
		m.flushCh = append(m.flushCh, flushCh)
		m.flushMutex.Unlock()
		go func(s driver.Driver, flushCh chan struct{}) {
			defer m.wg.Done()

			// Create the ticker
			m.l.RLock()
			ticker := time.NewTicker(m.flushInterval)
			m.l.RUnlock()

			// Send metrics until the context is canceled
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					m.sendRegisters(s)
				case <-flushCh:
					m.sendRegisters(s)
				}
			}
		}(s, flushCh)
	}

	m.wg.Wait()
	log.Debug("[metrics] stopped")
}

func (m *manager) sendRegisters(d driver.Driver) {
	var toSend []*driver.Registry
	m.l.RLock()
	for _, register := range m.registers {
		toSend = append(toSend, register)
	}
	m.l.RUnlock()

	if len(toSend) == 0 {
		log.Debug("no registry to send")
		return
	}

	go func() {
		err := d.Send(toSend)
		if err != nil {
			log.Error("[metrics] failed to send metrics")
		}
	}()
}

func (m *manager) addRegistry(name string, r metrics.Registry, tags map[string]string) {
	m.l.Lock()
	defer m.l.Unlock()

	m.registers[registryID(name, tags)] = &driver.Registry{
		Name:     name,
		Registry: r,
		Tags:     tags,
	}
}

func (m *manager) setFlushInterval(d time.Duration) {
	m.l.Lock()
	defer m.l.Unlock()

	m.flushInterval = d
}

func (m *manager) deleteRegistry(name string, tags map[string]string) error {
	m.l.Lock()
	defer m.l.Unlock()

	if _, exists := m.registers[registryID(name, tags)]; !exists {
		return ErrNotRegistered
	}

	delete(m.registers, name)
	return nil
}

func (m *manager) flush() {
	m.flushMutex.Lock()
	defer m.flushMutex.Unlock()

	for _, flushCh := range m.flushCh {
		select {
		case flushCh <- struct{}{}:
		default:
		}
	}
}

func (m *manager) stop() {
	m.cancel()
}

func registryID(name string, tags map[string]string) string {
	var tagIDs []string

	for tagName, tagValue := range tags {
		tagIDs = append(tagIDs, tagName+"="+tagValue)
	}
	sort.Strings(tagIDs)

	return fmt.Sprintf("%s[%s]", name, strings.Join(tagIDs, ","))
}

func registryName(id string) string {
	if idx := strings.Index(id, "["); idx != -1 {
		return id[:idx]
	}
	return id
}
