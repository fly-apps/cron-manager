# Cron Manager

Cron Manager is designed to enhance the way you manage Cron tasks on Fly.io.


## Getting started

Follow these steps to get your Cron Manager application up and running on Fly.io:

**1. Clone the project**
```bash
git clone git@github.com:fly-apps/cron-manager.git && cd cron-manager
```

**2. Create your app (Make sure the app name matches the fly.toml entry)**
```
fly apps create cron-manager
```

**3. Set your **FLY_API_TOKEN** as a secret**
```bash
fly secrets set FLY_API_TOKEN=$(fly auth token)
```

**4. Deploy your app**
```bash
fly deploy .
```


## Managing Schedules

Schedules can be defined using the `schedules.json` file.  Here's an example:

```json
[
    {
        "name": "uptime-check",
        "app_name": "my-app-name",
        "schedule": "* * * * *",
        "region": "iad",
        "command": "uptime",
        "config": {
            "auto_destroy": true,
            "disable_machine_autostart": true,
            "guest": {
                "cpu_kind": "shared",
                "cpus": 1,
                "memory_mb": 512
            },
            "image": "ghcr.io/livebook-dev/livebook:0.11.4",
            "restart": {
                "max_retries": 1,
                "policy": "no"
            }
        }
    }
]
```

**Note: The full `config` spec can be found within the 
[Machine API Spec](https://docs.machines.dev/#tag/machines/post/apps/{app_name}/machines).**


## Viewing Schedules
To view your registered schedules, you can use the `cm schedules list` command.  


```
fly ssh console --app-name <app-name>

cm schedules list
```

Output example:
```bash
|----|---------------|-----------------------------------------------|-----------|--------|----------------|---------|
| ID | TARGET APP    | IMAGE                                         | SCHEDULE  | REGION | RESTART POLICY | COMMAND |
|----|----------------|-----------------------------------------------|-----------|--------|----------------|---------|
| 1  | my-example-app | ghcr.io/livebook-dev/livebook:0.11.4          | * * * * * | iad    | no             | uptime  |
| 2  | my-example-app | docker-hub-mirror.fly.io/library/nginx:latest | 0 * * * * | ord    | no             | df -h   |
|----|----------------|-----------------------------------------------|-----------|--------|----------------|---------|
```

## Viewing Scheduled Jobs
Each job execution is recorded within a local sqlite.  To view the job history of a specific schedule, run the following command:

```bash
fly ssh console --app-name <app-name>
cm jobs list 1
```

Output example:
```bash
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


