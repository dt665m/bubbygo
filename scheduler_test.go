package bubbygo

import (
	"context"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	var (
		maxGoRoutines       = 2
		goRoutineQueueSize  = 1
		permanentGoRoutines = 1
		sched               = NewScheduler(maxGoRoutines, goRoutineQueueSize, permanentGoRoutines)
		jobSleep            = time.Duration(2 * time.Second)
	)

	//should spawn first go routine
	err := sched.Do(context.Background(), func() {
		time.Sleep(jobSleep)
		t.Log("finished job 1")
	})
	if err != nil {
		t.Fatalf("scheduling failed, expected %v, got %v", nil, err)
	}

	//should spawn second go routine
	err = sched.Do(context.Background(), func() {
		time.Sleep(jobSleep)
		t.Log("finished job 2")
	})
	if err != nil {
		t.Fatalf("scheduling failed, expected %v, got %v", nil, err)
	}

	//job should be queued
	done := make(chan struct{}, 1)
	err = sched.Do(context.Background(), func() {
		time.Sleep(jobSleep)
		t.Log("finished job 3")
		done <- struct{}{}
	})
	if err != nil {
		t.Fatalf("scheduling failed, expected %v, got %v", nil, err)
	}

	//job should timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err = sched.Do(ctx, func() {
		t.Log("finished job 4")
	})
	if err != context.DeadlineExceeded {
		t.Fatalf("scheduling failed, expected %v, got %v", context.DeadlineExceeded, err)
	}

	//block until job3 is done
	<-done
}

func TestJobCounter(t *testing.T) {
	const (
		maxGoRoutines       = 10
		goRoutineQueueSize  = 1000
		permanentGoRoutines = 10
	)
	sched := NewScheduler(maxGoRoutines, goRoutineQueueSize, permanentGoRoutines)

	//spin wait on jobs to finish
	start := time.Now()
	scheduledJobs := sched.Len()
	t.Log("jobs:", scheduledJobs)
	for scheduledJobs != 0 && time.Since(start) < 5*time.Second {
		scheduledJobs = sched.Len()
		if scheduledJobs == 0 {
			break
		}

		t.Log("waiting for %d jobs to finish\n", scheduledJobs)
		time.Sleep(100 * time.Millisecond)
	}
	t.Log("order processor stopped")
}
