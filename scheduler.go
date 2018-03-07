// Package bubbygo, a go routine pooling service to bound go routine spawning.
package bubbygo

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	DefaultKeepAlive = 30 * time.Second
)

// Scheduler is a bubbygo go routine scheduler
type Scheduler struct {
	goRoutines uint64
	tokens     chan struct{}
	queue      chan func()
	keepAlive  time.Duration
}

// NewScheduler creates a new bubbygo scheduler with default Go Routine KeepAlive Time of 30 seconds
func NewScheduler(maxRoutines, queueSize, preStart int) *Scheduler {
	if preStart <= 0 && queueSize > 0 {
		panic(fmt.Sprintf("scheduler deadlock settings detected, preStart: %d queueSize: %d", preStart, queueSize))
	}
	if preStart > maxRoutines {
		panic("preStart cannot be greater than maxRoutines")
	}
	s := &Scheduler{
		goRoutines: 0,
		tokens:     make(chan struct{}, maxRoutines),
		queue:      make(chan func(), queueSize),
		keepAlive:  time.Duration(DefaultKeepAlive),
	}

	for i := 0; i < preStart; i++ {
		s.tokens <- struct{}{}
		go s.process(func() {}, nil)
	}

	return s
}

// GoRoutineCount gives an estimate of the number of go routines active.  Do not use as form of synchronization
func (s *Scheduler) GoRoutineCount() int {
	return int(s.goRoutines)
}

// SetKeepAlive duration used to set the keepAlive time of generated go routines
func (s *Scheduler) SetKeepAlive(duration time.Duration) {
	s.keepAlive = duration
}

// Do consumes a token slot in s.tokens and starts a go routine to process the job if s.token isn't full.
// If s.tokens channel is full, the job is pushed into the s.queue until s.queue is full
// If both s.tokens and s.queue are full, Do will block on ctx.Done()
func (s *Scheduler) Do(ctx context.Context, job func()) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.queue <- job:
		return nil
	case s.tokens <- struct{}{}:
		go s.process(job, time.After(s.keepAlive))
		return nil
	}
}

// process handles the job closure passed in and dequeues jobs off the Scheduler.Queue until
// Scheduler.keepAlive duration.  process will stay alive indefinitely if keepAlive is a nil channel
func (s *Scheduler) process(job func(), keepAlive <-chan time.Time) {
	//handle the job that started this go routine
	atomic.AddUint64(&s.goRoutines, 1)
	job()

	//try to process more jobs before finishing, since spinning up go routines aren't free
RecycleGoRoutine:
	for {
		select {
		case job := <-s.queue:
			job()
		case <-keepAlive:
			break RecycleGoRoutine
		}

	}
	<-s.tokens

	atomic.AddUint64(&s.goRoutines, ^uint64(0))
}
