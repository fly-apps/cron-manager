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
  --command 'uptime'  
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
In the event you want to remove a registered schedule, you can simply run the `unregister` while specifying the associated schedule id.
```bash
cm schedules unregister 3
```

## Viewing Scheduled Jobs
Each job execution is recorded within a local sqlite.  To view the job history of a specific schedule, run the following command:

```bash
cm jobs list 3
|----|-----------|-----------|-------------------------|-------------------------|-------------------------|
| ID | STATUS    | EXIT CODE | CREATED AT              | UPDATED AT              | FINISHED AT             |
|----|-----------|-----------|-------------------------|-------------------------|-------------------------|
| 10 | completed | 0         | 2024-04-11 19:43:01 UTC | 2024-04-11 19:43:03 UTC | 2024-04-11 19:43:03 UTC |
| 9  | completed | 0         | 2024-04-11 19:42:01 UTC | 2024-04-11 19:42:03 UTC | 2024-04-11 19:42:03 UTC |
| 8  | completed | 0         | 2024-04-11 19:41:01 UTC | 2024-04-11 19:41:04 UTC | 2024-04-11 19:41:04 UTC |
| 7  | completed | 0         | 2024-04-11 19:40:01 UTC | 2024-04-11 19:40:03 UTC | 2024-04-11 19:40:03 UTC |
| 6  | completed | 0         | 2024-04-11 19:39:01 UTC | 2024-04-11 19:39:04 UTC | 2024-04-11 19:39:04 UTC |
| 5  | completed | 0         | 2024-04-11 19:38:01 UTC | 2024-04-11 19:38:04 UTC | 2024-04-11 19:38:04 UTC |
| 4  | completed | 0         | 2024-04-11 19:37:01 UTC | 2024-04-11 19:37:03 UTC | 2024-04-11 19:37:03 UTC |
| 3  | completed | 0         | 2024-04-11 19:36:01 UTC | 2024-04-11 19:36:03 UTC | 2024-04-11 19:36:03 UTC |
| 2  | completed | 0         | 2024-04-11 19:35:01 UTC | 2024-04-11 19:35:03 UTC | 2024-04-11 19:35:03 UTC |
|----|-----------|-----------|-------------------------|-------------------------|-------------------------|
```

**Note: stdout and stderr is also captured, but is not currently specified within the output.  To view this you must inspect the sqlite database located at `/data/state.db`**


