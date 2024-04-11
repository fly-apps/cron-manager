package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// main is the entry point for the application.
func main() {
	var rootCmd = &cobra.Command{Use: "cm"}
	var schedulesCmd = &cobra.Command{Use: "schedules"}
	var jobsCmd = &cobra.Command{Use: "jobs"}
	rootCmd.AddCommand(schedulesCmd)
	rootCmd.AddCommand(jobsCmd)

	schedulesCmd.AddCommand(listCmd)
	jobsCmd.AddCommand(listJobsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
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
		table.SetHeader([]string{"ID", "Image", "Schedule", "Restart Policy", "Command"})

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
