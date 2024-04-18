# Cron Manager

Cron Manager is designed to enhance the way you manage Cron jobs on Fly.io.

## Key Features and Benefits

### Isolated execution

Each job runs in its own isolated machine, preventing issues such as configuration drift, accumulation of temporary files, or other residual effects that could impact subsequent job executions. This isolation ensures that the outcome of one job does not negatively influence another, maintaining the integrity and reliability of each job.

### Centralized Scheduling

Manage all your Cron jobs centrally with a simple JSON configuration. This approach removes the need to embed cron dependencies within each production environment, streamlining setup and modifications. The use of a version-controlled configuration file enhances maintainability and auditability of scheduling changes.

### Simplified updates

Machines dedicated to specific Cron jobs are ephemeral and do not require updates. Any modifications to the schedules.json file will automatically be applied the next time the machine is provisioned for a scheduled job. This eliminates the need for ongoing maintenance of job environments, resulting in a more efficient update process.

### Enhanced Logs and Monitoring

Operating separate machines for each job greatly simplifies monitoring and auditing. This setup allows for straightforward tracking of the outcomes and logs of individual jobs, facilitating easier debugging and performance analysis.


## Getting started

Follow these steps to get your Cron Manager application up and running on Fly.io:

**Clone the project**
```bash
git clone git@github.com:fly-apps/cron-manager.git && cd cron-manager
```

**Create your app (Make sure the app name matches the fly.toml entry)**
```
fly apps create <new app name>
```

**Set your **FLY_API_TOKEN** as a secret**
```bash
fly secrets set FLY_API_TOKEN=$(fly auth token)
```

**Deploy your app**
```bash
fly deploy .
```


## Managing Schedules

Schedules are managed using the `schedules.json` file located within the projects root directory. Any new additions, updates, or deletions to this file are automatically reconciled on deploy.

### JSON Fields

- **`name`**: A unique identifier for the schedule. This is used to differentiate new schedules from schedules that need to be updated or deleted.
  **WARNING: Changing the `name` value after it has been deployed will result in the schedule being deleted and recreated. All historical job references for that schedule will be lost!**

- **`app_name`**: The name of your existing application that the schedule is associated with.  Provisoned Machines associated with each Job will be associated with this App.

- **`schedule`**: The cron expression that defines how often the Job should run. The format follows the standard cron format (minute, hour, day of month, month, day of week).

- **`region`**: The region where the scheduled job will execute.

- **`command`**: The command that will be executed from the provisioned Machine associated with the job.

- **`command_timeout`**: The total amount of time "in seconds" allowed for the command to execute. Default: 30 seconds

- **`enabled`**: A convenience flag that allows you to enable or disable a given schedule. When set to false, the schedule will not trigger any new jobs, but any existing job data will remain unaltered.

- **`config`**: A nested object containing the jobs Machine configuration. See the [Machine Config Spec](https://docs.machines.dev/#tag/machines/post/apps/{app_name}/machines) for more information.


### Example Schedule
```json
[
    {
        "name": "uptime-check",
        "app_name": "my-app-name",
        "schedule": "* * * * *",
        "region": "iad",
        "command": "uptime",
        "command_timeout": 30,
        "enabled": true,
        "config": {
            "metadata": {
                "fly_process_group": "cron"
            },
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



## Viewing Schedules
To view your registered schedules, you can use the `cm schedules list` command.

```bash
cm schedules list
```

Output example:
```bash
|----|------------------|-----------------------------------------------|-----------|--------|----------|---------|
| ID | TARGET APP       | IMAGE                                         | SCHEDULE  | REGION | COMMAND  | ENABLED |
|----|------------------|-----------------------------------------------|-----------|--------|----------|---------|
| 1  | my-example-app   | ghcr.io/livebook-dev/livebook:0.11.4          | * * * * * | iad    | sleep 10 | true    |
| 2  | my-example-app-2 | docker-hub-mirror.fly.io/library/nginx:latest | 0 * * * * | ord    | df -h    | false   |
|----|------------------|-----------------------------------------------|-----------|--------|----------|---------|
```

## Viewing Scheduled Jobs
Each job execution is recorded within a local sqlite. To view the job history of a specific schedule, ssh into the Machine and run the following:

```bash
cm jobs list <schedule-id>
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

## Viewing a Specific Job
```bash
cm jobs show <job-id>
```

Output example:
```
Job Details
  ID          = 30
  Status      = completed
  Machine ID  = 2866e19a795908
  Exit Code   = 0
  Created At  = 2024-04-15 14:34:01 UTC
  Updated At  = 2024-04-15 14:34:03 UTC
  Finished At = 2024-04-15 14:34:03 UTC
  Stdout      = 14:34:03 up 0 min,  0 user,  load average: 0.00, 0.00, 0.00
  Stderr      =
```


## Triggering Off-schedule Jobs
In the event you would like to trigger a Job "off schedule" for testing, you can do so with the `trigger` command.

```bash
cm jobs trigger <schedule-id>
```


