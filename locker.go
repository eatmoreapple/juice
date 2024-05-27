package juice

import "sync"

// RWLocker is a interface that can be used to lock、unlock and read lock、read unlock.
type RWLocker interface {
	RLock()
	RUnlock()
	Lock()
	Unlock()
}

type RWMutex = sync.RWMutex

var _ RWLocker = (*RWMutex)(nil)

// NoOpRWMutex is a no-op implementation of RWLocker.
type NoOpRWMutex struct{}

func (l *NoOpRWMutex) RLock() {}

func (l *NoOpRWMutex) RUnlock() {}

func (l *NoOpRWMutex) Lock() {}

func (l *NoOpRWMutex) Unlock() {}

var _ RWLocker = (*NoOpRWMutex)(nil)
