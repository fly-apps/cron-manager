package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/adhocore/gronx"
	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "cm"}
	var schedulesCmd = &cobra.Command{Use: "schedules"}
	var jobsCmd = &cobra.Command{Use: "jobs"}
	rootCmd.AddCommand(schedulesCmd)
	rootCmd.AddCommand(jobsCmd)

	schedulesCmd.AddCommand(listCmd)
	schedulesCmd.AddCommand(registerScheduleCmd)
	schedulesCmd.AddCommand(unregisterScheduleCmd)

	jobsCmd.AddCommand(listJobsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	registerScheduleCmd.Flags().StringP("app-name", "a", "", "The name of the app the job should run against")
	registerScheduleCmd.Flags().StringP("image", "i", "", "The image the Machine will run")
	registerScheduleCmd.Flags().StringP("schedule", "s", "", "The schedule to the job will run on. (Uses the cron format)")
	registerScheduleCmd.Flags().StringP("restart-policy", "r", "", "The restart policy for the Machine. (no, always, on-failure)")
	registerScheduleCmd.Flags().StringP("command", "c", "", "The command to run on the Machine")
	registerScheduleCmd.MarkFlagRequired("app-name")
	registerScheduleCmd.MarkFlagRequired("image")
	registerScheduleCmd.MarkFlagRequired("schedule")
	registerScheduleCmd.MarkFlagRequired("command")
}

var registerScheduleCmd = &cobra.Command{
	Use:   "register -app-name <app-name> -image <image> -schedule <schedule> -restart-policy <restart-policy> -command <command>",
	Short: "Register a new schedule",
	Long:  `Register a new schedule`,
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		appName, err := cmd.Flags().GetString("app-name")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		image, err := cmd.Flags().GetString("image")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schedule, err := cmd.Flags().GetString("schedule")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		gron := gronx.New()
		if gron.IsValid(schedule) == false {
			fmt.Println("Invalid schedule")
			os.Exit(1)
		}

		restartPolicy, err := cmd.Flags().GetString("restart-policy")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if restartPolicy == "" {
			restartPolicy = "no"
		}

		if restartPolicy != "no" && restartPolicy != "always" && restartPolicy != "on-failure" {
			fmt.Println("Invalid restart policy. Must be one of: no, always, on-failure")
			os.Exit(1)
		}

		command, err := cmd.Flags().GetString("command")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := store.CreateCronJob(appName, image, schedule, command, restartPolicy); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := cron.SyncCrontab(store); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Cronjob registered successfully")
	},
}

var unregisterScheduleCmd = &cobra.Command{
	Use:   "unregister <cronjob id>",
	Short: "Unregisters an existing schedule",
	Long:  `Unregisters an existing schedule`,
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		cronJobID := args[0]

		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := store.DeleteCronJob(cronJobID); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := cron.SyncCrontab(store); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Cronjob successfully unregistered")
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all registered cronjobs",
	Long:  `Lists all registered cronjobs`,
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cronjobs, err := store.ListCronJobs()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Target App", "Image", "Schedule", "Restart Policy", "Command"})

		// Set table alignment, borders, padding, etc. as needed
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetBorder(true) // Set to false to hide borders
		table.SetCenterSeparator("|")
		table.SetColumnSeparator("|")
		table.SetRowSeparator("-")
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderLine(true) // Enable header line
		table.SetAutoWrapText(false)

		for _, cj := range cronjobs {
			table.Append([]string{
				strconv.Itoa(cj.ID),
				fmt.Sprint(cj.AppName),
				fmt.Sprint(cj.Image),
				fmt.Sprint(cj.Schedule),
				fmt.Sprint(cj.RestartPolicy),
				fmt.Sprint(cj.Command),
			})
		}

		table.Render()
	},
}

var listJobsCmd = &cobra.Command{
	Use:   "list <cronjob id>",
	Short: "Lists all jobs for a specified schedule",
	Long:  `Lists all jobs for a specified schedule`,
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		cronJobID := args[0]

		store, err := cron.NewStore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		jobs, err := store.ListJobs(cronJobID, 10)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Status", "Exit Code", "Created At", "Updated At", "Finished At"})

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
