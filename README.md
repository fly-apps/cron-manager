# Cron Manager

Cron Manager is designed to enhance the way you manage Cron tasks on Fly.io.


## Getting started

**Clone the project**
```bash
git clone git@github.com:fly-apps/cron-manager.git && cd cron-manager
```

**Launch the project**

No tweaks should be needed to get started.
```bash
fly launch
```

**Set the FLY_API_TOKEN secret**
```bash
fly secrets set FLY_API_TOKEN=$(fly auth token)
```

If the Machine is in a `stopped` state at this point, you should go ahead and manually start it.

```bash
fly machines start <machine-id>
```


## Creating Schedules

**SSH into the Machine**
```
fly ssh console
```

**Use the `cm` cli to register your schedules**

```bash 
cm schedules register \
  --app-name shaun-pg-flex \
  --image 'ghcr.io/livebook-dev/livebook:0.11.4' \
  --schedule '* * * * *' \
  --command 'uptime' \
  --region 'ord'
```

## Viewing Schedules
To view your registered schedules, you can use the `cm schedules list` command.

```bash 
cm schedules list
|----|---------------|--------------------------------------|-----------|----------------|---------|
| ID | TARGET APP    | IMAGE                                | SCHEDULE  | RESTART POLICY | COMMAND |
|----|---------------|--------------------------------------|-----------|----------------|---------|
| 3  | shaun-pg-flex | ghcr.io/livebook-dev/livebook:0.11.4 | * * * * * | no             | uptime  |
|----|---------------|--------------------------------------|-----------|----------------|---------|
```

## Unregistering Schedules
In the event you would like to remove a specific schedule, you can simply run the `unregister` command while specifying the target schedule id.
```bash
cm schedules unregister 3
```

## Viewing Scheduled Jobs
Each job execution is recorded within a local sqlite.  To view the job history of a specific schedule, run the following command:

```bash
cm jobs list 3
|----|----------------|-----------|-----------|-------------------------|-------------------------|-------------------------|
| ID | MACHINE ID     | STATUS    | EXIT CODE | CREATED AT              | UPDATED AT              | FINISHED AT             |
|----|----------------|-----------|-----------|-------------------------|-------------------------|-------------------------|
| 30 | 185710da967398 | completed | 0         | 2024-04-11 20:03:01 UTC | 2024-04-11 20:03:03 UTC | 2024-04-11 20:03:03 UTC |
| 29 | 2865d32b356008 | completed | 0         | 2024-04-11 20:02:01 UTC | 2024-04-11 20:02:03 UTC | 2024-04-11 20:02:03 UTC |
| 28 | 683d67eb056e48 | completed | 0         | 2024-04-11 20:01:01 UTC | 2024-04-11 20:01:04 UTC | 2024-04-11 20:01:04 UTC |
| 27 | 080e07df930d08 | completed | 0         | 2024-04-11 20:00:01 UTC | 2024-04-11 20:00:06 UTC | 2024-04-11 20:00:06 UTC |
| 26 | 784e475f51d3e8 | completed | 0         | 2024-04-11 19:59:01 UTC | 2024-04-11 19:59:03 UTC | 2024-04-11 19:59:03 UTC |
| 25 | 28749e0bd51e68 | completed | 0         | 2024-04-11 19:58:01 UTC | 2024-04-11 19:58:03 UTC | 2024-04-11 19:58:03 UTC |
| 24 | e82de70b46ed78 | completed | 0         | 2024-04-11 19:57:01 UTC | 2024-04-11 19:57:03 UTC | 2024-04-11 19:57:03 UTC |
| 23 | d891745b3527d8 | completed | 0         | 2024-04-11 19:56:01 UTC | 2024-04-11 19:56:04 UTC | 2024-04-11 19:56:04 UTC |
| 22 | 7811372b1e2e68 | completed | 0         | 2024-04-11 19:55:01 UTC | 2024-04-11 19:55:03 UTC | 2024-04-11 19:55:03 UTC |
| 21 | 185710da967698 | completed | 0         | 2024-04-11 19:54:01 UTC | 2024-04-11 19:54:04 UTC | 2024-04-11 19:54:04 UTC |
|----|----------------|-----------|-----------|-------------------------|-------------------------|-------------------------|
```

**Note: stdout and stderr is also captured, but is not currently specified within the output.  To view this you must inspect the sqlite database located at `/data/state.db`**


