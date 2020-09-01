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
supports the fetching of go-modules within private repositories through build arguments

```bash
--build-arg user=${user} --build-arg personal_access_token=${personal_access_token}
```

## docker push

```bash
docker push ${GCP_GCR_HOSTNAME}/${GCP_PROJECT_ID}/executor:{VERSION}
```
