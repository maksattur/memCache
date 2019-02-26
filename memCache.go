package main

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Cache struct {
	sync.RWMutex
	defaultExpiration time.Duration
	cleanupInterval time.Duration
	items map[string]Item
}

type Item struct {
	Value interface{}
	Created time.Time
	Expiration int64
}

func New(defaultExpiration, cleanupInterval time.Duration) *Cache{
	items:=make(map[string]Item)
	cache := Cache{
		defaultExpiration:defaultExpiration,
		cleanupInterval:cleanupInterval,
		items:items,
	}
	if cleanupInterval >0 {
		cache.StartGC()
	}

	return &cache
}

func (c *Cache) Set(key string, value interface{}, duration time.Duration){
	var expiration int64

	if duration == 0{
		duration = c.defaultExpiration
	}

	if duration > 0{
		expiration = time.Now().Add(duration).UnixNano()
	}

	c.Lock()
	defer c.Unlock()
	c.items[key] = Item{
		Value:value,
		Created:time.Now(),
		Expiration:expiration,
	}
}

func (c *Cache) Get(key string) (interface{}, bool)  {
	c.RLock()
	defer c.RUnlock()

	item, found := c.items[key]

	if !found{
		return nil,false
	}

	if item.Expiration > 0{
		if time.Now().UnixNano() > item.Expiration{
			return nil, false
		}
	}
	return item.Value, true
}

func (c *Cache) GetAll() map[string]interface{}  {
	c.RLock()
	defer c.RUnlock()
	allItems := make(map[string]interface{})
	for k, v :=range c.items{
		allItems[k] = v.Value
		//fmt.Println(k, " ", v)
	}
	return allItems
}

func (c *Cache) Delete(key string) error{
	c.Lock()
	defer c.Unlock()

	if _, found := c.items[key]; !found{
		return errors.New("Key not found")
	}
	delete(c.items, key)
	return nil
}

func (c *Cache) Count() (count int) {
	c.RLock()
	defer c.RUnlock()
	count = len(c.items)
	return
}

func (c *Cache) StartGC()  {
	go c.GC()
}

func (c *Cache) GC()  {
	for{
		<-time.After(c.cleanupInterval)
		fmt.Println("GC is started")
		if c.items == nil{
			return
		}
		if keys := c.expiredKeys(); len(keys) != 0{
			fmt.Println("We have expiredKeys: keys = ", keys)
			c.clearItems(keys)
		}
	}
}

func (c *Cache) expiredKeys() (keys []string){
	c.RLock()
	defer c.RUnlock()

	for k, i := range c.items{
		if time.Now().UnixNano() > i.Expiration && i.Expiration > 0{
			keys = append(keys, k)
		}
	}
	return
}

func (c *Cache) clearItems(keys []string)  {
	c.Lock()
	defer c.Unlock()
	for _, k := range keys{
		delete(c.items, k)
	}
}

const N  = 10

func main() {
	myCache := New(5 * time.Minute, 1 * time.Minute)

	fmt.Println("Init count = ", myCache.Count()) // 0

	go func() {
		for i:=1;i<=10;i++{
			myCache.Set(strconv.Itoa(i),i, time.Duration(i) * time.Minute)
		}
	}()

	go func() {
		for i:=11;i<=20;i++  {
			myCache.Set(strconv.Itoa(i),i, time.Duration(i-10) * time.Minute)
		}
	}()


	time.Sleep(3 * time.Second)
	allItems := myCache.GetAll()
	for k,v:=range allItems{
		fmt.Printf("key -> %s : val -> %v\n", k, v)
	}

	fmt.Scanln()
}

