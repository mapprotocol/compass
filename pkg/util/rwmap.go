package util

import "sync"

type RWMap struct {
	sync.RWMutex
	m map[string]int64
}

func NewRWMap() *RWMap {
	return &RWMap{
		m: make(map[string]int64, 0),
	}
}
func (m *RWMap) Get(k string) (int64, bool) {
	m.RLock()
	defer m.RUnlock()
	v, existed := m.m[k]
	return v, existed
}

func (m *RWMap) Set(k string, v int64) {
	m.Lock()
	defer m.Unlock()
	m.m[k] = v
}

func (m *RWMap) Delete(k string) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, k)
}

func (m *RWMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.m)
}

func (m *RWMap) Each(f func(k string, v int64) bool) {
	m.RLock()
	defer m.RUnlock()

	for k, v := range m.m {
		if !f(k, v) {
			return
		}
	}
}
