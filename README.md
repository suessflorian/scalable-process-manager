# Scalable Process Manager

Suppose your service needs to expose the ability handle the lifecycle of some in-deterministically long living process. Say it needs to _spawn_, _list_ and _stop_ processes.

The naive solution typically includes some utilisation of [`context.WithCancel`](https://pkg.go.dev/context#WithCancel), where a centralised `map[processID]context.CancelFunc` would host the means to cancel processes running on a detached goroutine.

The problem here is the introduction of service state.

# Solution

```sh
go run .
```
