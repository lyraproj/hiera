package lookup

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

	newValue := replacer(c.values[key])
	c.values[key] = newValue
	return newValue
}

// Delete deletes the given key from the map
func (c *ConcurrentMap) Delete(key string) {
	c.lock.Lock()
	delete(c.values, key)
	c.lock.Unlock()
}

// EnsureSet checks if the given key is set and returns it if that is the case. Otherwise
// it calls the producer and assigns the returned value. The produced value is then returned.
//
// The producer does not execute within a mutex.
func (c *ConcurrentMap) EnsureSet(key string, producer func() interface{})  interface{} {
	c.lock.RLock()
	oldValue, isSet := c.values[key]
	c.lock.RUnlock()
	if isSet {
		if l, ok := oldValue.(sync.RWMutex); ok {
			// The value is currently a lock. Wait for the real value
			l.RLock()
			oldValue, _ = c.values[key]
			l.RUnlock()
		}
		return oldValue
	}

	// Take the write lock
	c.lock.Lock()

	// Must check again, another thread might have intervened
	if oldValue, isSet = c.values[key]; isSet {
		c.lock.Unlock()
		if l, ok := oldValue.(sync.RWMutex); ok {
			// The value is currently a lock. Wait for the real value
			l.RLock()
			oldValue, _ = c.values[key]
			l.RUnlock()
		}
		return oldValue
	}

	// Replace the value with a RWMutex that is locked.
	lock := sync.RWMutex{}
	lock.Lock()
	defer func() {
		delete(c.values, key)
		lock.Unlock()
	}()
	c.values[key] = lock

	// Let go of the write lock
	c.lock.Unlock()

	// Call the producer. A deadlock will occur if this call results in a new lookup for the same key
	// but that's OK. The alternative (not using locks) would be an endless recursion
	value := producer()
	c.values[key] = value
	return value
}

// Get returns the value for the given key together with a bool to indicate
// if the key was found
func (c *ConcurrentMap) Get(key string) (value interface{}, ok bool) {
	c.lock.RLock()
	value, ok = c.values[key]
	c.lock.RUnlock()
	return
}

// Set adds or replaces the value for the given key
func (c *ConcurrentMap) Set(key string, value interface{}) {
	c.lock.Lock()
	c.values[key] = value
	c.lock.Unlock()
}

