// Package bubbygo, a go routine pooling service to bound go routine spawning.
package bubbygo

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	DefaultKeepAlive = 500 * time.Millisecond
)

// Scheduler is a bubbygo go routine scheduler
type Scheduler struct {
	goRoutines uint64
	sem        chan struct{}
	queue      chan func()
	keepAlive  time.Duration
}

// NewScheduler creates a new bubbygo scheduler
func NewScheduler(maxRoutines, queueSize, preStart int) *Scheduler {
	if preStart <= 0 && queueSize > 0 {
		panic(fmt.Sprintf("scheduler deadlock settings detected, preStart: %d queueSize: %d", preStart, queueSize))
	}
	if preStart > maxRoutines {
		panic("preStart cannot be greater than maxRoutines")
	}
	s := &Scheduler{
		goRoutines: 0,
		sem:        make(chan struct{}, maxRoutines),
		queue:      make(chan func(), queueSize),
		keepAlive:  time.Duration(DefaultKeepAlive),
	}

	for i := 0; i < preStart; i++ {
		s.sem <- struct{}{}
		go s.process(func() {}, false)
	}

	return s
}

// Len gives an estimate of the number of go routines active.  Do not use for thread synchronization
func (s *Scheduler) Len() int {
	return int(s.goRoutines)
}

// SetKeepAlive sets the keepAlive time of generated go routines
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
	case s.sem <- struct{}{}:
		go s.process(job, true)
		return nil
	}
}

// process handles the job closure passed in and dequeues jobs off the Scheduler.Queue until
// Scheduler.keepAlive duration.  process will stay alive indefinitely if keepAlive is a nil channel
func (s *Scheduler) process(job func(), willExpire bool) {
	// handle the job that started this go routine
	atomic.AddUint64(&s.goRoutines, 1)
	job()

	// try to process more jobs before finishing, since spinning up go routines aren't free
	var expire *time.Timer
	if willExpire {
		expire = time.NewTimer(s.keepAlive)
	}
RecycleGoRoutine:
	for {
		select {
		case job := <-s.queue:
			job()

			// avoid race condition where reset can still observe previous in-flight message
			if !expire.Stop() {
				<-expire.C
			}
			expire.Reset(s.keepAlive)
		case <-expire.C:
			break RecycleGoRoutine
		}
	}

	// finish and release semaphore
	<-s.sem
	atomic.AddUint64(&s.goRoutines, ^uint64(0))
}
