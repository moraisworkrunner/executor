# executor

receives work requests and issues responses to the notifier queue

## build and Run

```bash
go build
./executor
```

## docker build

```bash
docker build -t ${GCP_GCR_HOSTNAME}/${GCP_PROJECT_ID}/executor:{VERSION} -f Dockerfile .
```

### now a public repo

Previously, this repo and others in the org were private and required build arguments, which
is necessary within an organization protecting its source code. Therefore, the Dockerfile
supports the fetching of go-modules within private repositories through build arguments:

```bash
--build-arg user=${user} --build-arg personal_access_token=${personal_access_token}
```

## docker push

```bash
docker push ${GCP_GCR_HOSTNAME}/${GCP_PROJECT_ID}/executor:{VERSION}
```

## MAX_ATTEMPTS and TaskExecutionCount

Cloud Task queues have a retry capability such that this service need only return 2xx to declare the execution a success, or otherwise to retry. MAX_ATTEMPTS is an env variable assigned to the executor service coordinated with the task queue's max-attempts variable, which can be set like this:

```bash
gcloud tasks queues update incoming --max-attempts=7
```

In theory, if the environment-provided MAX_ATTEMPTS is equal to the current header value of `X-CloudTasks-TaskExecutionCount`, provided by the Cloud Tasks agent, upon a failure to process work, a Failure notification should be sent to the client via the notification task queue.

### Always 0

For some reason, the X-CloudTasks-TaskExecutionCount is always set to 0. It could be that a specific response code is neded to tell the queue to count each execution properly, or that it is a bug. Google does not seem to think so.. https://issuetracker.google.com/issues/148756284

A bug worth fixing!

## Scaling

When dealing with a finite queue and retries, inevitably one could face the issue where there are many failed requests being retried `X` times causing low throughput of likely-successful tasks until the issue subsides naturally.

This application addresses this by allowing for infrastructure-driven retry back-off through chaining of additional queues and/or executor services. Since it is known, via MAX_ATTEMPTS, when the number of tries for the current queue has been exhausted and the task will be removed, the same task can be handed to another queue with a lower retry count, or even handled by a separate executor service with different rate limits.

The nice thing about this solution is that it can be leveraged not only ahead of time to mitigate concerns by strategy, but it can also be used reactively to ongoing incidents.

Queues and services can be configured for larger/smaller bucketing, more/less instances, and higher/lower rate limits according to the category of work or problem description.

Known Use Cases:

* Chained executors: (incoming) -> (executor) -> (incoming-2) -> (executor-2) -> ...

## Background

This service was written for the sole purpose of trying out these Google technologies, and designed as such with sole intent of solving the scaling problem described in the previous section for this architecture.
