package impl

import "sync"

type ConcurrentMap struct {
	lock   sync.RWMutex
	values map[string]interface{}
}

// NewConcurrentMap creates a new map with the given capacity
func NewConcurrentMap(capacity int) *ConcurrentMap {
	return &ConcurrentMap{values: make(map[string]interface{}, capacity)}
}

// AtomicReplace replaces the value for the given key and returns the new value.
// The new value is produced  by the replacer function which gets the old value
// as a parameter.
//
// The replacer function executes within a mutex lock must not call back into the
// map. If it does, the thread will deadlock.
func (c *ConcurrentMap) AtomicReplace(key string, replacer func(oldValue interface{}) interface{}) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()
	oldValue, _ := c.valueMutexWait(key)

	newValue := replacer(oldValue)
	c.values[key] = newValue
	return newValue
}

// Delete deletes the given key from the map
func (c *ConcurrentMap) Delete(key string) {
	c.lock.Lock()
	if _, ok := c.valueMutexWait(key); ok {
		delete(c.values, key)
	}
	c.lock.Unlock()
}

func (c *ConcurrentMap) valueMutexWait(key string) (interface{}, bool) {
	for {
		oldValue, isSet := c.values[key]
		if isSet {
			if l, ok := oldValue.(*sync.RWMutex); ok {
				// The value is currently a lock. Wait until it's released
				c.lock.Unlock()
				l.RLock()
				c.lock.Lock()
				l.RUnlock()
				continue
			}
		}
		return oldValue, isSet
	}
}

// EnsureSet checks if the given key is set and returns it if that is the case. Otherwise
// it calls the producer and assigns the returned value. The produced value is then returned.
//
// The producer does not execute within a mutex.
func (c *ConcurrentMap) EnsureSet(key string, producer func() (interface{}, bool)) (value interface{}, ok bool) {
	// Take the write lock
	c.lock.Lock()

	value, ok = c.valueMutexWait(key)
	if ok {
		c.lock.Unlock()
		return value, true
	}

	// Replace the value with a RWMutex that is locked.
	lock := sync.RWMutex{}
	lock.Lock()

	defer func() {
		c.lock.Lock()
		if ok {
			c.values[key] = value
		}
		lock.Unlock()
		c.lock.Unlock()
	}()
	c.values[key] = &lock

	// Let go of the write lock
	c.lock.Unlock()

	// Call the producer. A deadlock will occur if this call results in a new lookup for the same key
	// but that's OK. The alternative (not using locks) would be an endless recursion
	value, ok = producer()
	return
}

// Get returns the value for the given key together with a bool to indicate
// if the key was found
func (c *ConcurrentMap) Get(key string) (value interface{}, ok bool) {
	c.lock.RLock()
	value, ok = c.values[key]
	c.lock.RUnlock()
	if ok {
		if l, isMutex := value.(*sync.RWMutex); isMutex {
			// The value is currently a lock. Wait for the real value
			l.RLock()
			value, ok = c.values[key]
			l.RUnlock()
		}
	}
	return
}

// Set adds or replaces the value for the given key
func (c *ConcurrentMap) Set(key string, value interface{}) {
	c.lock.Lock()
	if oldValue, isSet := c.values[key]; isSet {
		if l, ok := oldValue.(*sync.RWMutex); ok {
			// The value is currently a lock. Wait until it's released
			c.lock.Unlock()
			l.RLock()
			c.lock.Lock()
			l.RUnlock()
		}
	}
	c.values[key] = value
	c.lock.Unlock()
}
