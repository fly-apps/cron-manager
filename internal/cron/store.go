package cron

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/superfly/fly-go"
)

const (
	migrationsPath = "/usr/local/share/migrations"
	storePath      = "/data/state.db?_busy_timeout=5000&_journal_mode=WAL"
)

type Schedule struct {
	ID       int               `json:"id"`
	Name     string            `json:"name"`
	AppName  string            `json:"app_name"`
	Schedule string            `json:"schedule"`
	Command  string            `json:"command"`
	Region   string            `json:"region"`
	Config   fly.MachineConfig `json:"config"`
}

type Job struct {
	ID         int            `json:"id"`
	ScheduleID int            `json:"schedule_id"`
	Status     string         `json:"status"`
	Stdout     sql.NullString `json:"stdout"`
	Stderr     sql.NullString `json:"stderr"`
	MachineID  sql.NullString `json:"machine_id"`
	ExitCode   sql.NullInt64  `json:"exit_code"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	FinishedAt sql.NullTime   `json:"finished_at"`
}

type Store struct {
	*sql.DB
}

func NewStore() (*Store, error) {
	s, err := sql.Open("sqlite3", storePath)
	if err != nil {
		return nil, err
	}

	return &Store{s}, nil
}

func (s Store) SetupDB(log *logrus.Logger) error {
	migrations := &migrate.FileMigrationSource{
		Dir: migrationsPath,
	}

	n, err := migrate.Exec(s.DB, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("error applying migrations: %w", err)
	}

	log.Infof("applied %d migrations", n)

	return nil
}

func (s Store) FindSchedule(id int) (*Schedule, error) {
	var name, region, appName, schedule, command, config string
	row := s.QueryRow("SELECT name, region, app_name, schedule, command, config FROM schedules WHERE id = ?", id)
	if err := row.Scan(&name, &region, &appName, &schedule, &command, &config); err != nil {
		return &Schedule{}, err
	}

	var cfg fly.MachineConfig

	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return nil, err
	}

	return &Schedule{
		ID:       id,
		Name:     name,
		Region:   region,
		AppName:  appName,
		Schedule: schedule,
		Command:  command,
		Config:   cfg,
	}, nil
}

func (s Store) FindScheduleByName(name string) (*Schedule, error) {
	var id int
	var appName, schedule, command, region, config string
	row := s.QueryRow("SELECT id, app_name, schedule, command, region, config FROM schedules WHERE name = ?", name)
	if err := row.Scan(&id, &appName, &schedule, &command, &region, &config); err != nil {
		return nil, err
	}

	var cfg fly.MachineConfig

	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return nil, err
	}

	return &Schedule{
		ID:       id,
		Name:     name,
		AppName:  appName,
		Schedule: schedule,
		Command:  command,
		Region:   region,
		Config:   cfg,
	}, nil
}

func (s Store) ListSchedules() ([]Schedule, error) {
	rows, err := s.Query("SELECT id, name, app_name, schedule, region, command, config FROM schedules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var id int
		var name, appName, region, schedule, command, config string
		if err := rows.Scan(&id, &name, &appName, &schedule, &region, &command, &config); err != nil {
			return nil, err
		}

		var cfg fly.MachineConfig

		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return nil, err
		}

		schedules = append(schedules, Schedule{
			ID:       id,
			Name:     name,
			AppName:  appName,
			Schedule: schedule,
			Region:   region,
			Command:  command,
			Config:   cfg,
		})
	}

	return schedules, nil
}

func (s Store) FindJob(jobID string) (*Job, error) {
	var id int
	var status string
	var createdAt, updatedAt time.Time
	var exitCode sql.NullInt64
	var finishedAt sql.NullTime
	var machineID, stdout, stderr sql.NullString

	row := s.QueryRow("SELECT id, status, machine_id, exit_code, stdout, stderr, created_at, updated_at, finished_at FROM jobs where id = ?", jobID)
	if err := row.Scan(&id, &status, &machineID, &exitCode, &stdout, &stderr, &createdAt, &updatedAt, &finishedAt); err != nil {
		return &Job{}, err
	}

	return &Job{
		ID:         id,
		Status:     status,
		MachineID:  machineID,
		ExitCode:   exitCode,
		Stdout:     stdout,
		Stderr:     stderr,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		FinishedAt: finishedAt,
	}, nil
}

func (s Store) ListJobs(scheduleID string, limit int) ([]Job, error) {
	query := fmt.Sprintf("SELECT id, status, machine_id, exit_code, stdout, stderr, created_at, updated_at, finished_at FROM jobs where schedule_id = %s ORDER BY id DESC LIMIT %d", scheduleID, limit)
	rows, err := s.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var id int
		var createdAt, updatedAt time.Time
		var exitCode sql.NullInt64
		var finishedAt sql.NullTime
		var machineID, stdout, stderr sql.NullString

		var status string
		if err := rows.Scan(&id, &status, &machineID, &exitCode, &stdout, &stderr, &createdAt, &updatedAt, &finishedAt); err != nil {
			return nil, err
		}

		jobs = append(jobs, Job{
			ID:         id,
			Status:     status,
			MachineID:  machineID,
			ExitCode:   exitCode,
			Stdout:     stdout,
			Stderr:     stderr,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			FinishedAt: finishedAt,
		})
	}

	return jobs, nil

}

func (s Store) CreateSchedule(sch Schedule) error {
	insertScheduleSQL := `INSERT INTO schedules (name, app_name, schedule, command, region, config) VALUES (?, ?, ?, ?, ?, ?);`

	cfgBytes, err := json.Marshal(sch.Config)
	if err != nil {
		return fmt.Errorf("error marshalling machine config: %w", err)
	}

	_, err = s.Exec(insertScheduleSQL,
		sch.Name,
		sch.AppName,
		sch.Schedule,
		sch.Command,
		sch.Region,
		string(cfgBytes),
	)

	return err
}

func (s Store) UpdateSchedule(sch Schedule) error {
	cfgBytes, err := json.Marshal(sch.Config)
	if err != nil {
		return fmt.Errorf("error marshalling machine config: %w", err)
	}

	insertScheduleSQL := `UPDATE schedules SET app_name = ?, schedule = ?, command = ?, region = ?, config = ? WHERE name = ?;`
	_, err = s.Exec(insertScheduleSQL,
		sch.AppName,
		sch.Schedule,
		sch.Command,
		sch.Region,
		string(cfgBytes),
		sch.Name,
	)

	return err
}

func (s Store) DeleteSchedule(id string) error {
	deleteScheduleSQL := `DELETE FROM schedules WHERE id = ?;`
	_, err := s.Exec(deleteScheduleSQL, id)
	if err != nil {
		return fmt.Errorf("error deleting schedule: %w", err)
	}

	// Delete all jobs associated with the schedule
	deleteJobsSQL := `DELETE FROM jobs WHERE schedule_id = ?;`
	_, err = s.Exec(deleteJobsSQL, id)
	return err
}

func (s Store) CreateJob(scheduleID int) (int, error) {
	insertJobSQL := `INSERT INTO jobs (schedule_id, status, created_at, updated_at) VALUES (?, ?, ?, ?);`

	result, err := s.DB.Exec(insertJobSQL, scheduleID, "running", time.Now(), time.Now())
	if err != nil {
		return 0, fmt.Errorf("error executing insert job SQL: %w", err)
	}

	// Get the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting last insert ID: %w", err)
	}

	return int(id), nil
}

func (s Store) UpdateJobStatus(id int, status string) error {
	updateJobStatusSQL := `UPDATE jobs SET status = ?, updated_at = ? WHERE id = ?;`
	_, err := s.Exec(updateJobStatusSQL, status, time.Now(), id)
	return err
}

func (s Store) UpdateJobMachine(id int, machineID string) error {
	updateJobMachineSQL := `UPDATE jobs SET machine_id = ?, updated_at = ? WHERE id = ?;`
	_, err := s.Exec(updateJobMachineSQL, machineID, time.Now(), id)
	return err
}

func (s Store) UpdateJobOutput(id int, exitCode int, stdout, stderr string) error {
	updateJobOutputSQL := `UPDATE jobs SET exit_code = ?, stdout = ?, stderr = ?, updated_at = ? WHERE id = ?;`
	_, err := s.Exec(updateJobOutputSQL, exitCode, stdout, stderr, time.Now(), id)
	return err
}

func (s Store) FailJob(id int, exitCode int, stderr string) error {
	updateJobStatusSQL := `UPDATE jobs SET status = 'failed', exit_code = ?, stderr = ?, updated_at = ?, finished_at = ? WHERE id = ?;`
	_, err := s.Exec(updateJobStatusSQL, exitCode, stderr, time.Now(), time.Now(), id)
	return err
}

func (s Store) CompleteJob(id int, exitCode int, stdout string) error {
	updateJobStatusSQL := `UPDATE jobs SET status = 'completed', exit_code = ?, stdout = ?, updated_at = ?, finished_at = ? WHERE id = ?;`
	_, err := s.Exec(updateJobStatusSQL, exitCode, stdout, time.Now(), time.Now(), id)
	return err
}
