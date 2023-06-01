package juice

import "sync"

// RWLocker is a interface that can be used to lock、unlock and read lock、read unlock.
type RWLocker interface {
	RLock()
	RUnlock()
	Lock()
	Unlock()
}

var _ RWLocker = (*sync.RWMutex)(nil)
