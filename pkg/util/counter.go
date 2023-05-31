package util

import (
	"fmt"
	"sync"
)

type Counter struct {
	lock sync.Mutex
	data map[string]int
}

func NewCounter() *Counter {
	return &Counter{
		lock: sync.Mutex{},
		data: map[string]int{},
	}
}

func (c *Counter) Inc(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	v, ok := c.data[key]
	if ok {
		c.data[key] = v + 1
		return
	}
	c.data[key] = 0
}

func (c *Counter) DeleteKey(key string) {
	c.lock.Lock()
	defer c.lock.Lock()

	delete(c.data, key)
}

func (c *Counter) Counts(key string) (int, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	v, ok := c.data[key]
	if !ok {
		return 0, fmt.Errorf("cannot get key %s", key)
	}
	var err error = nil
	if v < 0 {
		err = fmt.Errorf("count %s:%d is negative", key, v)
	}
	return v, err
}

func (c *Counter) Dec(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	v, ok := c.data[key]
	if ok {
		if v > 1 {
			c.data[key] = v - 1
			return nil
		}
		if v == 1 {
			c.DeleteKey(key)
			return nil
		}
		return fmt.Errorf("cannot minus one: key %s has value %d", key, v)
	}
	return fmt.Errorf("cannot find key %s", key)
}
