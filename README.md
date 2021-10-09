# Scalable Process Manager

Suppose your service needs to expose the ability handle the lifecycle of some in-deterministically long living process. Say it needs to _spawn_, _list_ and _stop_ processes.

The naive solution typically includes some utilisation of [`context.WithCancel`](https://pkg.go.dev/context#WithCancel), where a centralised `map[processID]context.CancelFunc` would host the means to cancel processes running on a detached goroutine\*.

The problem here is that this `CancelFunc` map centralises process state, prohibiting horizontal scaling of the service as each service would own the means of cancelling different process. I personally ran into this problem. So this serves as a playground to explore alternative methods.

# Current Solution
Basically pushing the responsibility of tracking process state to the database. For each process, a goroutine is spawned, which concurrently polls for process cancel requests and runs the process. Use of [`nginx` for some simple load balancing](http://nginx.org/en/docs/http/load_balancing.html).

```sh
docker compose up --scale process-manager=12
```

\* see commit `5581706feca5f183198ec148042a5ea061ae9771` as an example
