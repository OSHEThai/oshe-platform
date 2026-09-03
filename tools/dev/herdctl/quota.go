package main

import (
	"errors"
	"sync"
)

var ErrQuotaExceeded = errors.New("zero-cost quota exceeded")
var ErrConcurrencyLimit = errors.New("concurrency limit exceeded")

type QuotaController struct {
	mu           sync.Mutex
	used         int
	limit        int
	active       int
	concurrency  int
	stopBehavior bool
}

func NewQuotaController(limit, concurrency int) *QuotaController {
	return &QuotaController{
		limit:       limit,
		concurrency: concurrency,
	}
}

func (q *QuotaController) SetStopBehavior(stop bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stopBehavior = stop
}

func (q *QuotaController) Acquire(cost int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.stopBehavior {
		return errors.New("system stopped")
	}

	if q.active >= q.concurrency {
		return ErrConcurrencyLimit
	}

	if q.used+cost > q.limit {
		return ErrQuotaExceeded
	}

	q.used += cost
	q.active++
	return nil
}

func (q *QuotaController) Release() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.active > 0 {
		q.active--
	}
}

func (q *QuotaController) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.used = 0
	q.active = 0
	q.stopBehavior = false
}
