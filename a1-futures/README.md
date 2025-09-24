# Assignment 1: Futures in Go

**Due: Fri, 12 Sep**

Please make sure to regularly commit and push your work to Github. As with all assignments in this course, 10% of the grade will come from the quality of your git commit history.

## Futures
In concurrent programming, a `Future` is an object that acts as a placeholder for a result that is initially unknown but will be computed asynchronously. In this library, we provide a `Future` type and function headers for a `Wait` function. The `Wait` function returns a slice containing the values for the Futures passed into it, once the Futures are `Completed`.

We will describe how to implement this in more detail later in this document.

## Motivation
You might be wondering why we need to implement such a library, when we can achieve the same in a naive fashion. To demonstrate this, we provide you with a file [naive/naive_requests.go](naive/naive_requests.go). This implements a slow `GetWeatherData` function and its driver code to collect temperature readings in a "non-Future" manner.

You can run this using the following command:
```shell
go run naive/naive_requests.go
```

`GetWeatherData` is non-deterministic, since it returns a random output, and its internal delays are randomized, you might see a different output on every run. 

Some example outputs are:
```shell
Naive Requests Demo
Timing out 250ms
Total true values received: 4
```
```shell
Naive Requests Demo
Done
Total true values received: 6
```

You should carefully read the code and the corresponding comments explaining the logic.

## Your Task
1) Your first task is to implement the `Wait` function, which will wait for the first `n` futures to return or until a specified `timeout` occurs. You will also implement logic to filter the results using a provided `filter` function. More details about the parameters and returns of `Wait` are below.

2) Once you have a basic `Wait` function in place, you need to implement a user defined function that uses Futures. Specifically, you would be implementing the `GetWeatherData` with Futures support built in. You will find the function headers in [future.go](future.go). You will also implement a simple `filter` function called `heatWaveWarning` that only keeps temperatures over 35 degrees.
   
`GetWeatherData` should perform the same tasks it did in the previous assignment, with the only difference being that your implementation of should return a completable `Future` immediately, and then, in the background, work on actually getting the data. 

`GetWeatherData` will call the `CompleteFuture()` function once it's finished processing. The result of the `GetWeatherData` will then be accessible via calling `GetResult()` on the same `Future` object.

The tester expects a `Future` object as the return value of `GetWeatherData` and will fail if you don't do so.

Note that you must ensure that `Wait` returns a slice of all the results from the completed futures. 

You will be using this library as part of a later assignment, so make sure that you start early so that you have enough time to weed out all possible bugs.

### Parameters
`futures []*Future`: A slice of `Future` objects to wait on.

`n int`: The number of futures to wait for.

`timeout time.Duration`: The maximum time to wait before timing out.

`filter func(interface{}) bool`: A function to check each future’s result. Returns true if the result meets the required condition. `filter` can be `nil`, in which case all results should be kept.

### Returns
`[]interface{}`: A slice of results from the first n futures that completed and met the condition specified by `filter`.

## Observations
Take a look at the code required to get the values from `GetWeatherData` in [naive_requests.go](naive/naive_requests.go).

You will notice that with the library you just implemented, the 20-ish lines of code in the naive approach are reduced to the following in [future_test.go](future_test.go)!

```go
results := make([]*Future, 0)
nPeers := 10
for i := 0; i < nPeers; i++ {
  results = append(results, slowFunction(false, false))
}

returned_results := Wait(results, (nPeers/2)+1, 300*time.Millisecond, nil)
```

Pretty cool, right?

## Testing your code
You can test your code by running the following command:
```
go test -v -race
```

You can also run individual test cases by running:
```
go test -v -race -run TestFutureBasic
```
Replace "TestFutureBasic" with the name of the test you're trying to run.

On successful completion, you should see something similar to this output
```
=== RUN   TestFutureBasic
--- PASS: TestFutureBasic (0.19s)
=== RUN   TestFutureTimeout
--- PASS: TestFutureTimeout (0.30s)
=== RUN   TestFutureFilter
--- PASS: TestFutureFilter (0.27s)
=== RUN   TestFutureUnreliable
--- PASS: TestFutureUnreliable (0.22s)
=== RUN   TestGetWeatherDataBasic
--- PASS: TestGetWeatherDataBasic (1.21s)
=== RUN   TestGetWeatherDataDelayedResponses
--- PASS: TestGetWeatherDataDelayedResponses (1.20s)
=== RUN   TestGetWeatherDataOneFail
--- PASS: TestGetWeatherDataOneFail (1.20s)
=== RUN   TestGetWeatherDataFilter
--- PASS: TestGetWeatherDataFilter (1.20s)
PASS
ok      cs351/a2-futures        6.813s
```

## Submission

Upload the `future.go` file to Gradescope. Please do not change any of the function signatures, or the autograder may fail to run.

All the best!
