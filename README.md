# BubbyGo

BubbyGo provides a simple go routine scheduler optimized for short living jobs that are mostly idle, such as a socket service delivering push-style data.  Since go routines are unbounded by default and allocates memory, a process that can have an unbounded number of requests generating go routines will have unpredictable/uncontrollable failure characteristics.

The BubbyGo scheduler is designed to spin up go routines to handle jobs as function closures based on a set of rules.  The number of Go Routines allowed to be generated through the scheduler is fixed.  After all go routines are scheduled, jobs are pushed to a queue until the queue becomes full.  At this point, it is up to the caller to decide to block and wait or use a go context.Context to timeout/cancel.  Scheduler Spawned go routines will stay alive and process jobs off the queue so as not to spin up more unnecessarily.

```go
var(
    maxGoRoutines = 1000
    goRoutineQueueSize = 50
    permanentGoRoutines = 1;
    sched = bubbygo.NewScheduler(maxGoRoutines, goRoutineQueueSize, permanentGoRoutines)
)

//set keepalive of goroutines to 5 minutes
sched.SetKeepAlive(time.Duration(5 *time.Minute))

//block until the scheduled job is done
done := make(chan struct{}, 1)
err := sched.Do(context.Background(), func() {
    time.Sleep(5 * time.Second)
    fmt.Println("finished!")
    done<-struct{}{}
})
<-done
```

# KeepAlive
By default, the Scheduler reuses Go Routines for 30 seconds by processing queued jobs.  This can be tuned with Scheduler.SetKeepAlive(time.duration).  The purposes of keeping the go routines alive is so they can serve extra jobs on the queue without shutting down and respinning more go routines up.  This setting is a performance tradeoff, mainly used to fine tune the expected sustained load of requests.  If the expected jobs are short and bursty, use a lower KeepAlive.