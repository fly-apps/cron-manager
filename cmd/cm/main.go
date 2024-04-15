package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var rootCmd = &cobra.Command{Use: "cm"}
	var schedulesCmd = &cobra.Command{Use: "schedules"}
	var jobsCmd = &cobra.Command{Use: "jobs"}
	rootCmd.AddCommand(schedulesCmd)
	rootCmd.AddCommand(jobsCmd)

	schedulesCmd.AddCommand(syncCrontabCmd)
	schedulesCmd.AddCommand(listCmd)

	jobsCmd.AddCommand(listJobsCmd)
	jobsCmd.AddCommand(processJobCmd)
	jobsCmd.AddCommand(showJobCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var log = logrus.New()

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered schedules",
	Long:  `List all registered schedules`,
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schedules, err := store.ListSchedules()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Target App", "Image", "Schedule", "Region", "Restart Policy", "Command"})

		// Set table alignment, borders, padding, etc. as needed
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetBorder(true) // Set to false to hide borders
		table.SetCenterSeparator("|")
		table.SetColumnSeparator("|")
		table.SetRowSeparator("-")
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderLine(true) // Enable header line
		table.SetAutoWrapText(false)

		for _, schedule := range schedules {
			table.Append([]string{
				strconv.Itoa(schedule.ID),
				fmt.Sprint(schedule.AppName),
				fmt.Sprint(schedule.Config.Image),
				fmt.Sprint(schedule.Schedule),
				fmt.Sprint(schedule.Region),
				fmt.Sprint(schedule.Config.Restart.Policy),
				fmt.Sprint(schedule.Command),
			})
		}

		table.Render()
	},
}

var processJobCmd = &cobra.Command{
	Use:   "trigger <schedule id>",
	Short: "Triggers a job for the specified schedule",
	Long:  `Triggers a job for the specified schedules. `,
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		// Convert the schedule ID to an integer
		scheduleID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schedule, err := store.FindSchedule(scheduleID)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := cron.ProcessJob(cmd.Context(), log, store, schedule.ID); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
var listJobsCmd = &cobra.Command{
	Use:   "list <schedule id>",
	Short: "Lists all jobs for the specified schedule",
	Long:  `Lists all jobs for the specified schedule`,
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		scheduleID := args[0]

		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		jobs, err := store.ListJobs(scheduleID, 10)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Machine ID", "Status", "Exit Code", "Created At", "Updated At", "Finished At"})

		// Set table alignment, borders, padding, etc. as needed
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetBorder(true) // Set to false to hide borders
		table.SetCenterSeparator("|")
		table.SetColumnSeparator("|")
		table.SetRowSeparator("-")
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderLine(true) // Enable header line
		table.SetAutoWrapText(false)

		for _, j := range jobs {
			created := j.CreatedAt.Format("2006-01-02 15:04:05 UTC")
			updated := j.UpdatedAt.Format("2006-01-02 15:04:05 UTC")
			var finished string
			if j.FinishedAt.Valid {
				finished = j.FinishedAt.Time.Format("2006-01-02 15:04:05 UTC")
			} else {
				finished = ""
			}

			table.Append([]string{
				strconv.Itoa(j.ID),
				j.MachineID.String,
				fmt.Sprint(j.Status),
				strconv.Itoa(int(j.ExitCode.Int64)),
				created,
				updated,
				finished,
			})
		}

		table.Render()
	},
}

var showJobCmd = &cobra.Command{
	Use:   "show <job id>",
	Short: "Show job details",
	Long:  `Show job details`,
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		jobID := args[0]

		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		job, err := store.FindJob(jobID)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetBorder(false)
		table.SetAutoWrapText(false)
		table.SetColumnSeparator("=")
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

		var finishedAt string
		if job.FinishedAt.Valid {
			finishedAt = job.FinishedAt.Time.Format("2006-01-02 15:04:05 UTC")
		} else {
			finishedAt = ""
		}

		rows := [][]string{
			{
				strconv.Itoa(job.ID),
				job.Status,
				job.MachineID.String,
				strconv.Itoa(int(job.ExitCode.Int64)),
				job.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
				job.UpdatedAt.Format("2006-01-02 15:04:05 UTC"),
				finishedAt,
				job.Stdout.String,
				job.Stderr.String,
			},
		}

		cols := []string{
			"ID",
			"Status",
			"Machine ID",
			"Exit Code",
			"Created At",
			"Updated At",
			"Finished At",
			"Stdout",
			"Stderr",
		}

		fmt.Println("Job Details")

		for _, row := range rows {
			for i, col := range cols {
				table.Append([]string{col, row[i]})
			}
			table.Render()
		}
	},
}

var syncCrontabCmd = &cobra.Command{
	Use:   "sync",
	Short: "Syncs sqlite schedules with crontab",
	Long:  `Syncs sqlite schedules with crontab`,
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := cron.SyncSchedules(store, log); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Crontab synced successfully")
	},
}
