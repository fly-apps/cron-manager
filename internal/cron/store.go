package cron

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/superfly/fly-go"
)

const (
	StorePath      = "/data/state.db?_busy_timeout=5000&_journal_mode=WAL"
	MigrationsPath = "/usr/local/share/migrations"

	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)

type Schedule struct {
	ID       int               `json:"id" db:"id"`
	Name     string            `json:"name" db:"name"`
	AppName  string            `json:"app_name" db:"app_name"`
	Schedule string            `json:"schedule" db:"schedule"`
	Command  string            `json:"command" db:"command"`
	Region   string            `json:"region" db:"region"`
	Config   fly.MachineConfig `json:"config" db:"config"`
}

// TODO - Remove this
type RawSchedule struct {
	ID       int    `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	AppName  string `json:"app_name" db:"app_name"`
	Schedule string `json:"schedule" db:"schedule"`
	Command  string `json:"command" db:"command"`
	Region   string `json:"region" db:"region"`
	Config   string `json:"config" db:"config"` // JSON string
}

type Job struct {
	ID         int            `json:"id" db:"id"`
	ScheduleID int            `json:"schedule_id" db:"schedule_id"`
	Status     string         `json:"status" db:"status"`
	Stdout     sql.NullString `json:"stdout" db:"stdout"`
	Stderr     sql.NullString `json:"stderr" db:"stderr"`
	MachineID  sql.NullString `json:"machine_id" db:"machine_id"`
	ExitCode   sql.NullInt64  `json:"exit_code" db:"exit_code"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
	FinishedAt sql.NullTime   `json:"finished_at" db:"finished_at"`
}

type Store struct {
	*sqlx.DB
}

func NewStore(storePath string) (*Store, error) {
	s, err := sqlx.Open("sqlite3", storePath)
	if err != nil {
		return nil, err
	}

	return &Store{s}, nil
}

func (s Store) SetupDB(log *logrus.Logger, migrationDirPath string) error {
	migrations := &migrate.FileMigrationSource{
		Dir: migrationDirPath,
	}

	n, err := migrate.Exec(s.DB.DB, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("error applying migrations: %w", err)
	}

	log.Infof("applied %d migrations", n)

	return nil
}

func (s Store) FindSchedule(id int) (*Schedule, error) {
	var rawSchedule RawSchedule
	if err := s.DB.Get(&rawSchedule, "SELECT * FROM schedules WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("error getting schedule: %w", err)
	}

	return convertToStandardSchedule(rawSchedule)
}

func (s Store) FindScheduleByName(name string) (*Schedule, error) {
	var rawSchedule RawSchedule
	if err := s.DB.Get(&rawSchedule, "SELECT * FROM schedules WHERE name = ?", name); err != nil {
		return nil, fmt.Errorf("error getting schedule: %w", err)
	}

	return convertToStandardSchedule(rawSchedule)
}

func (s Store) ListSchedules() ([]Schedule, error) {
	var rawSchedules []RawSchedule
	if err := s.DB.Select(&rawSchedules, "SELECT * FROM schedules"); err != nil {
		return nil, fmt.Errorf("error getting schedules: %w", err)
	}

	var schedules []Schedule
	for _, raw := range rawSchedules {
		schedule, err := convertToStandardSchedule(raw)
		if err != nil {
			return nil, fmt.Errorf("error converting schedule: %w", err)
		}
		schedules = append(schedules, *schedule)
	}

	return schedules, nil
}

func (s Store) FindJob(jobID string) (*Job, error) {
	var job Job
	if err := s.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", jobID); err != nil {
		return nil, fmt.Errorf("error getting job: %w", err)
	}

	return &job, nil
}

func (s Store) ListJobs(scheduleID string, limit int) ([]Job, error) {
	var jobs []Job
	if err := s.DB.Select(&jobs, "SELECT * FROM jobs WHERE schedule_id = ? ORDER BY id DESC LIMIT ?", scheduleID, limit); err != nil {
		return nil, fmt.Errorf("error getting jobs: %w", err)
	}

	return jobs, nil
}

func (s Store) ListReconcilableJobs() ([]Job, error) {
	var jobs []Job
	if err := s.DB.Select(&jobs, "SELECT * FROM jobs WHERE status IN (?,?)", JobStatusPending, JobStatusRunning); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s Store) CreateSchedule(sch Schedule) error {
	cfgBytes, err := json.Marshal(sch.Config)
	if err != nil {
		return fmt.Errorf("error marshalling machine config: %w", err)
	}

	_, err = s.DB.Exec("INSERT INTO schedules (name, app_name, schedule, command, region, config) VALUES (?, ?, ?, ?, ?, ?)",
		sch.Name,
		sch.AppName,
		sch.Schedule,
		sch.Command,
		sch.Region,
		sch.Config,
		cfgBytes,
	)

	return err
}

func (s Store) UpdateSchedule(sch Schedule) error {
	cfgBytes, err := json.Marshal(sch.Config)
	if err != nil {
		return fmt.Errorf("error marshalling machine config: %w", err)
	}

	_, err = s.DB.Exec("UPDATE schedules SET app_name = ?, schedule = ?, command = ?, region = ?, config = ? WHERE name = ?",
		sch.AppName,
		sch.Schedule,
		sch.Command,
		sch.Region,
		cfgBytes,
		sch.Name,
	)

	return err
}

func (s Store) DeleteSchedule(id string) error {
	_, err := s.Exec("DELETE FROM schedules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("error deleting schedule: %w", err)
	}

	// Delete all jobs associated with the schedule
	_, err = s.Exec("DELETE FROM jobs WHERE schedule_id = ?", id)
	return err
}

func (s Store) CreateJob(scheduleID int) (int, error) {
	result, err := s.DB.Exec("INSERT INTO jobs (schedule_id, status, created_at, updated_at) VALUES ($1, $2, $3, $4)",
		scheduleID,
		JobStatusPending,
		time.Now(),
		time.Now(),
	)

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
	_, err := s.Exec("UPDATE jobs SET status = ?, updated_at = ? WHERE id = ?",
		status,
		time.Now(),
		id,
	)
	return err
}

func (s Store) UpdateJobMachine(id int, machineID string) error {
	_, err := s.Exec("UPDATE jobs SET machine_id = ?, updated_at = ? WHERE id = ?",
		machineID,
		time.Now(),
		id,
	)
	return err
}

func (s Store) SetJobResult(id int, exitCode int, stdout, stderr string) error {
	_, err := s.Exec("UPDATE jobs SET exit_code = ?, stdout = ?, stderr = ?, updated_at = ? WHERE id = ?",
		exitCode,
		stdout,
		stderr,
		time.Now(),
		id,
	)
	return err
}

func (s Store) FailJob(id int, exitCode int, stderr string) error {
	_, err := s.Exec("UPDATE jobs SET status = ?, exit_code = ?, stderr = ?, updated_at = ?, finished_at = ? WHERE id = ?",
		JobStatusFailed,
		exitCode,
		stderr,
		time.Now(),
		time.Now(),
		id,
	)
	return err
}

func (s Store) CompleteJob(id int, exitCode int, stdout string) error {
	_, err := s.Exec("UPDATE jobs SET status = ?, exit_code = ?, stdout = ?, updated_at = ?, finished_at = ? WHERE id = ?",
		JobStatusCompleted,
		exitCode,
		stdout,
		time.Now(),
		time.Now(),
		id,
	)
	return err
}

func convertToStandardSchedule(raw RawSchedule) (*Schedule, error) {
	var cfg fly.MachineConfig
	if err := json.Unmarshal([]byte(raw.Config), &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &Schedule{
		ID:       raw.ID,
		Name:     raw.Name,
		AppName:  raw.AppName,
		Schedule: raw.Schedule,
		Command:  raw.Command,
		Region:   raw.Region,
		Config:   cfg,
	}, nil
}
