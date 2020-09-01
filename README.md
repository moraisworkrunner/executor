# executor

receives work requests and issues responses to the notifier queue

## build and Run

```bash
go build
./executor
```

## docker build

```bash
docker build -t ${GCP_GCR_HOSTNAME}/${GCP_PROJECT_ID}/executor:{VERSION} -f Dockerfile . --build-arg user=${user} --build-arg personal_access_token=${personal_access_token}
```

## docker push

```bash
docker push ${GCP_GCR_HOSTNAME}/${GCP_PROJECT_ID}/executor:{VERSION}
```
