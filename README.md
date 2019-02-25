# ForgeRock SaaS Software Engineer Coding Challenge

## Introduction and Overview

I have created a demo program that shows a microservices-based solution for tracking and reporting events of interest using the suggested method of Redis pub/sub along with the keyspace notifications for these events.  For the demo, I have focused on a couple of performance metrics (latency for fulfilling a client request, and number of successes and failures).  These events occur in one microservice, a geolocator service that depends on the US Census Bureau's free service, and are relayed via the keyspace notifications to another microservice, the analyzer.  The analyzer is mostly a stub at this point, although it performs a few basic functions such as computing the average latency over the latest set of lookup calls.  In real life, I'd envision this microservice connecting to something like a database or system like graylog to graph service performance.

While the assignment suggested it was ok to simply have some workers that sleep and do nothing, I felt that for the sake of proper integration testing and monitoring, I would build REST APIs around both services.  Since both services (and Redis) are running in Docker containers, having a way of poking into the system from the outside is essential for testing and monitoring.  And having an HTTP already, it was not too big of a deal to make it actually do something.

One other quick note about "workers".  In Go, every HTTP request is served in it's own goroutine, so these are my workers.  I had considered making each HTTP request handle bulk requests via a pool of goroutines, and while this eminently doable, it's seems unnecessary to demonstrate the behavior.  I have a multi-threaded HTTP-based integration test that effectively creates this set of "workers" and the system functions normally under those conditions.

## Events and Semantics
One of the interesting aspects of this project is considering the semantics of the data passed and the resulting requirements for the event generator (locator) and receiver (analyzer).  If the data is stateless and the order is not particularly important, then every worker (HTTP request goroutine here) can send it's data at once, and the event receiver can be mutithreaded.  Stuff being averaged or aggregated, such as the things I am sending, fits this bill.  That said, I have written the code so that the receiver can be multi-threaded or not (set numWorkers to 1 for the latter).  The go-redis package I used provides a channel to receive subscribed events, so this is perfect for single or multiple receiver workers.

On the sending side, there are some situations where you might want to ensure that certain sets of events are sent in a bunch, so that concurrent threads do not intersperse setting the same keys.  For this, I have added the capability of the sender acquiring a redis-wide lock by writing a (crappy) lock using the (crappy) algorithm suggested in the Redis doc for "Set".  This algorithm has many issues, but it seems to work ok for small databases when properly configured.  It is turned off, as it's not needed for my data, but I have also successfully run my integration test with it on.  There are proper distributed locking algorithms out there based on redigo, but I have only worked with go-redis to date, and am not super-experienced with Redis, so I stuck with what I knew best.

Back to semantics, Redis has the feature that multiple requests from all over are queued up and served in order.  Using that along with locking senders to ensure atomicity of sent data, a receiver can make sense of the data when received in a single receivng goroutine.  But there's still no guarantee that by the time the receiver has read the data, that the key's value hasn't been changed since the event was received.  If the latter semantics are important, the sender would need to receive some kind of ack back before sending more data (probably using another keyspace event), which I did not get to implementing, but it would not be hard to add in the locking mechanism.  On the other hand, using unique keys for distinct pieces of data of a given type might be a better way of keeping the data straight, as opposed to a complex locking mechanism.

## Addressing the Requirements

* Build instructions (any 3rd party requirements and how to generally get them setup on either linux or mac) -- docker-compose or something similar is suggested

I have written a docker-compose script and Dockerfiles to build both microservices.  There is also a puny Makefile present, so you can run `make composeup` to build and start both services and redis.  You can ctrl-C to kill everything, or much better you can run `make composedown`, which more actually seems to send the termination signal to PID 1.  If you don't have `make`, you can simply copy those commands out of the Makefile.  Running compuseup from a terminal, you will see the output from each service, a different color at the start of the output for each service.  Note I created vendor folder for each service, to avoid `go get`

- Usage instructions (i.e. samples to actually show how it works)
The integration test under _locator/tests/integration_ invokes the REST API to do gecode lookups and get the resultant stats from the analyzer, including concurrent execution.  You run this from your external enviornment outside the container (MacOS for me), simply run `go test` in that directory.

Note the docker containers all expose system chosen external ports, so you can use a tool like Postman to test the endpoints.

```
$ docker ps
CONTAINER ID        IMAGE                   COMMAND                  CREATED             STATUS              PORTS                     NAMES
f44629980651        locator-demo_locator    "./locator"              10 seconds ago      Up 8 seconds        0.0.0.0:32933->8080/tcp   locator-demo_locator_1
f03f41d3e485        locator-demo_analyzer   "./analyzer"             10 seconds ago      Up 8 seconds        0.0.0.0:32932->8090/tcp   locator-demo_analyzer_1
faea84450839        redis:alpine            "docker-entrypoint.s…"   17 hours ago        Up 9 seconds        0.0.0.0:32931->6379/tcp   locator-demo_redis_1
```

Here we see the services are running on localhost on ports 32933 and 32932.

Assuming those port numbers, in Postman you could do a POST of a lookup request to the locator:
```
http://localhost:32933/v1/lookup
```
with payload:
```
{
    "struct_number": "4600",
    "street": "Silver Hill Rd",
    "city": "Suitland",
    "state": "MD",
    "zip": "20746"
}
```

and get back
```
{
    "zip": "20746",
    "coordinates": {
        "x": -76.92691,
        "y": 38.846542
    }
}
```

Then you could invoke the statistics endpoint of the analyzer woith a GET:
```
http://localhost:32913/v1/statistics
```
and see something like
```
{
    "success": 1,
    "failure": 0,
    "latency_events": 1,
    "latency": "1.851453233s"
}
```

- Unit tests

There are unit tests in some packages.  The ones in locator/geolocator/geolocator_test.go show my preferred style of creating an array of test case structs, followed by test logic for each array member.  If I had more time, I would have mocked out the actual service lookup in addition to the Store interface.
- Integration tests
Again, locator/tests.integration, run `go test`

- Are there any shortcomings of the code?
As mentioned, the distributed locking mechanism is weak.  All the threading models of sender and receiver are all available and configurable, but as I said earlier, it might be nice to have an ack keysapce event receiver on the sender side, but I can add this to the code if requested.

- How might this project be scaled?
The locator can easily be replicated N times.  To scale the analyzer (the receiver) is more challenging, again it depends on the semantics of the data being sent and atomicity requirements.  The redis-go package does not support things like pipelines concurrently, for one.  The specfific answer depends on the exact requirements.

- How might one approach doing sequential versus parallel tasks?
I discussed this in the "Events and Semantics" seciton above.

## Code Roadmap

The root directory is _locator-demo_, both services _locator_ and _analyzer_ are directly below this.  Each has a main that launches the entities of interest.  Both also contain the Redis initialization code, and thanalyzer has extra code for pubsub and keyspace listeners.

In _locator_ the package `geolocator` has the code that sets the Redis keys that will be picked up by the analyzer.  Given tht I wrote the code to potentially use another mechanism besides redis, the geolocator only knows about the generic _Store_ interface.  The Redis-specific store in under the `store` package.

Some lines to check out:
[https://github.com/gdotgordon/locator-demo/blob/master/locator/geolocator/geolocator.go#L68]
[https://github.com/gdotgordon/locator-demo/blob/master/locator/geolocator/geolocator.go#L159]
[https://github.com/gdotgordon/locator-demo/blob/master/locator/store/store.go]

In _analyzer_ the receiver processes the received events based on the tags received.
[https://github.com/gdotgordon/locator-demo/blob/master/analyzer/receiver/receiver.go]

## External Pckages Used and Licenses
* https://github.com/go-redis/redis - BSD 2-Clause "Simplified" License
* https://github.com/tidwall/gjson - MIT License
* https://github.com/gorilla/mux - BSD 3-Clause "New" or "Revised" License
* https://github.com/rs/xid - MIT License

## Conclusion
Being fairly new to Redis, there are possibly more advanced features that would have helped my implmentation.  I did my best in the time I had to add enough flexibility to consider the various scenarios, but it writing one-size-fits-all code is not always the best approach, and if this were production code, I would have fleshed out each of the semantic case and written a framework to handle each of these.

In general, hopefully the code shows good principles of microservice architecture and design and testing.

Back to our regularly scheduled program ...


## Project Background

We have recently decided that all new projects will be done in Typescript for front-end work and Golang for back-end and utility work.  We obviously understand that not every solution will fit into those two buckets, but any outliers will be addressed on a case-by-case basis.  This coding challenge is meant to be non-trivial example of a problem we recently worked on originally in nodejs and then in golang.

Often times we will have tasks that need to be executed (some in parallel and some in sequential order -- in our case allocating cloud resources).  These tasks could take a few seconds or a few minutes and may or may not fail in various ways.

## The Challenge

We are asking that you write a simple worker pattern type project using a modern language. We prefer golang but if you are more comfortable in another language, we are also polyglot and will accept a language you are most proficient with.  These workers can be stubs (i.e. they can just sleep for random periods of time -- bonus points for doing an actual task) but they should feed back into the system in some way -- i.e. tracking and reporting.  We suggest using redis for both the pub/sub worker interactions as well as datastore -- but the technology choice is yours as long as we can build it and it fits a similar pattern.

This challenge is not meant to be a production ready version, but it is meant to show thought process, concepts, and problem solving skills.  You are free to use any 3rd party libraries or code to accomplish the task as long as it is LICENSED accordingly.

## Requirements / Concepts

Some requirements / concepts we would like to see (either in code or at least outlined in the README):

- Build instructions (any 3rd party requirements and how to generally get them setup on either linux or mac) -- docker-compose or something similar is suggested
- Usage instructions (i.e. samples to actually show how it works)
- Unit tests
- Integration tests
- Are there any shortcomings of the code?
- How might this project be scaled?
- How might one approach doing sequential versus parallel tasks?

## Completion Format

- [ ] To do this challenge, please Fork this repository to your own public github repository.
- [ ] Build all the things, and have fun! 
- [ ] Check your code into the fork when you are complete.
- [ ] :shipit:  Send the recruiter at ForgeRock the link to your repository.  

We will review what you did during one of your interview sessions!

