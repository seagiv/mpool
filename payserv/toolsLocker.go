package main

import (
	"fmt"
	"strings"
	"sync"
)

var txLocker *locker

type locker struct {
	m     sync.Mutex
	items map[string]*sync.Mutex
}

func getLockNameTx(ID int64) string {
	return fmt.Sprintf("tx_%d", ID)
}

func getLockNameCoin(coinID int64) string {
	return fmt.Sprintf("coin_%d", coinID)
}

func newLocker() *locker {
	return &locker{items: make(map[string]*sync.Mutex)}
}

//Lock -TODO-
func (l *locker) Lock(key string) {
	l.m.Lock()

	ll := l.items[key]

	if ll == nil {
		ll = &sync.Mutex{}

		l.items[key] = ll
	}

	l.m.Unlock()

	ll.Lock()
}

//Unlock -TODO-
func (l *locker) Unlock(key string) {
	if l.items[key] == nil {
		return
	}

	l.items[key].Unlock()
}

//Get -TODO-
func (l *locker) Get(key string) *sync.Mutex {
	return l.items[key]
}

//Delete -TODO-
func (l *locker) Delete(key string) {
	l.m.Lock()
	defer l.m.Unlock()

	if l.items[key] != nil {
		l.items[key].Unlock()

		delete(l.items, key)
	}
}

//List -TODO-
func (l *locker) List() string {
	l.m.Lock()
	defer l.m.Unlock()

	var r []string

	for k := range l.items {
		r = append(r, k)
	}

	return strings.Join(r, " ")
}
