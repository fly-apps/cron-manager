package cron

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	storePath = "/data/state.db?_busy_timeout=5000&_journal_mode=WAL"
)

type Store struct {
	*sql.DB
}

func (s Store) SetupDB() error {
	createCronJobsTableSQL := `CREATE TABLE IF NOT EXISTS cronjobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_name TEXT NOT NULL,
		image TEXT,
		schedule TEXT NOT NULL,
		command TEXT NOT NULL,
		restart_policy TEXT CHECK(restart_policy IN ('no', 'always', 'on-failure')) NOT NULL DEFAULT 'on-failure',
		UNIQUE(app_name, image)
	);`
	_, err := s.Exec(createCronJobsTableSQL)
	if err != nil {
		return err
	}

	createJobsTableSQL := `CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cronjob_id INTEGER NOT NULL,
		status TEXT CHECK(status IN ('pending', 'running', 'completed', 'failed')) NOT NULL DEFAULT 'pending',
		machine_id TEXT,
		exit_code INTEGER,
		stdout TEXT,
		stderr TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		FOREIGN KEY(cronjob_id) REFERENCES cronjobs(id)
	);`
	_, err = s.Exec(createJobsTableSQL)
	if err != nil {
		return err
	}

	return nil
}

func NewStore() (*Store, error) {
	s, err := sql.Open("sqlite3", storePath)
	if err != nil {
		return nil, err
	}

	return &Store{s}, nil
}

type CronJob struct {
	ID            int    `json:"id"`
	AppName       string `json:"app_name"`
	Image         string `json:"image"`
	Schedule      string `json:"schedule"`
	Command       string `json:"command"`
	RestartPolicy string `json:"restart_policy"`
}

type Job struct {
	ID         int            `json:"id"`
	CronJobID  int            `json:"cronjob_id"`
	Status     string         `json:"status"`
	Stdout     sql.NullString `json:"stdout"`
	Stderr     sql.NullString `json:"stderr"`
	MachineID  sql.NullString `json:"machine_id"`
	ExitCode   sql.NullInt64  `json:"exit_code"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	FinishedAt sql.NullTime   `json:"finished_at"`
}

func (s Store) FindCronJob(id int) (*CronJob, error) {
	var appName, image, schedule, command, restartPolicy string
	row := s.QueryRow("SELECT app_name, image, schedule, command, restart_policy FROM cronjobs WHERE id = ?", id)
	if err := row.Scan(&appName, &image, &schedule, &command, &restartPolicy); err != nil {
		return &CronJob{}, err
	}

	return &CronJob{
		ID:            id,
		AppName:       appName,
		Image:         image,
		Schedule:      schedule,
		Command:       command,
		RestartPolicy: restartPolicy,
	}, nil
}

func (s Store) ListCronJobs() ([]CronJob, error) {
	rows, err := s.Query("SELECT id, app_name, image, schedule, command, restart_policy FROM cronjobs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cronJobs []CronJob
	for rows.Next() {
		var id int
		var appName, image, schedule, command, restartPolicy string
		if err := rows.Scan(&id, &appName, &image, &schedule, &command, &restartPolicy); err != nil {
			return nil, err
		}

		cronJobs = append(cronJobs, CronJob{
			ID:            id,
			AppName:       appName,
			Image:         image,
			Schedule:      schedule,
			Command:       command,
			RestartPolicy: restartPolicy,
		})
	}

	return cronJobs, nil
}

func (s Store) CreateCronJob(appName, image, schedule, command, restartPolicy string) error {
	insertCronJobSQL := `INSERT INTO cronjobs (app_name, image, schedule, command, restart_policy) VALUES (?, ?, ?, ?, ?);`
	_, err := s.Exec(insertCronJobSQL, appName, image, schedule, command, restartPolicy)
	return err
}

func (s Store) CreateJob(cronjobID int) (int, error) {
	insertJobSQL := `INSERT INTO jobs (cronjob_id, status, created_at, updated_at) VALUES (?, ?, ?, ?);`

	result, err := s.DB.Exec(insertJobSQL, cronjobID, "running", time.Now(), time.Now())
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

func (s Store) FindJob(id int) (*Job, error) {
	var cronJobID int
	var status string
	var createdAt, updatedAt time.Time
	var machineID, stdout, stderr sql.NullString
	var finishedAt sql.NullTime

	row := s.QueryRow("SELECT cronjob_id, machine_id, status, stdout, stderr, created_at, updated_at, finished_at FROM jobs WHERE id = ?", id)
	if err := row.Scan(&cronJobID, &machineID, &status, &stdout, &stderr, &createdAt, &updatedAt, &finishedAt); err != nil {
		return &Job{}, err
	}

	return &Job{
		ID:         id,
		CronJobID:  cronJobID,
		MachineID:  machineID,
		Status:     status,
		Stdout:     stdout,
		Stderr:     stderr,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		FinishedAt: finishedAt,
	}, nil
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
