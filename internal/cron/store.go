package cron

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	storePath = "/data/state.db?_busy_timeout=5000&_journal_mode=WAL"
)

type Store struct {
	*sql.DB
}

func (s Store) SetupDB() error {
	createSchedulesTableSQL := `CREATE TABLE IF NOT EXISTS schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_name TEXT NOT NULL,
		image TEXT,
		schedule TEXT NOT NULL,
		region TEXT NOT NULL,
		command TEXT NOT NULL,
		restart_policy TEXT CHECK(restart_policy IN ('no', 'always', 'on-failure')) NOT NULL DEFAULT 'on-failure',
		UNIQUE(app_name, image)
	);`
	_, err := s.Exec(createSchedulesTableSQL)
	if err != nil {
		return err
	}

	createJobsTableSQL := `CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		schedule_id INTEGER NOT NULL,
		status TEXT CHECK(status IN ('pending', 'running', 'completed', 'failed')) NOT NULL DEFAULT 'pending',
		machine_id TEXT,
		exit_code INTEGER,
		stdout TEXT,
		stderr TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		FOREIGN KEY(schedule_id) REFERENCES schedules(id)
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

type Schedule struct {
	ID            int    `json:"id"`
	AppName       string `json:"app_name"`
	Image         string `json:"image"`
	Schedule      string `json:"schedule"`
	Command       string `json:"command"`
	Region        string `json:"region"`
	RestartPolicy string `json:"restart_policy"`
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

func (s Store) FindSchedule(id int) (*Schedule, error) {
	var appName, image, schedule, command, restartPolicy string
	row := s.QueryRow("SELECT app_name, image, schedule, command, restart_policy FROM schedules WHERE id = ?", id)
	if err := row.Scan(&appName, &image, &schedule, &command, &restartPolicy); err != nil {
		return &Schedule{}, err
	}

	return &Schedule{
		ID:            id,
		AppName:       appName,
		Image:         image,
		Schedule:      schedule,
		Command:       command,
		RestartPolicy: restartPolicy,
	}, nil
}

func (s Store) ListSchedules() ([]Schedule, error) {
	rows, err := s.Query("SELECT id, app_name, image, schedule, region, command, restart_policy FROM schedules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var id int
		var appName, region, image, schedule, command, restartPolicy string
		if err := rows.Scan(&id, &appName, &image, &schedule, &region, &command, &restartPolicy); err != nil {
			return nil, err
		}

		schedules = append(schedules, Schedule{
			ID:            id,
			AppName:       appName,
			Image:         image,
			Schedule:      schedule,
			Region:        region,
			Command:       command,
			RestartPolicy: restartPolicy,
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

type CreateScheduleRequest struct {
	AppName       string `json:"app_name"`
	Image         string `json:"image"`
	Schedule      string `json:"schedule"`
	Command       string `json:"command"`
	Region        string `json:"region"`
	RestartPolicy string `json:"restart_policy"`
}

func (s Store) CreateSchedule(req CreateScheduleRequest) error {
	insertScheduleSQL := `INSERT INTO schedules (app_name, image, schedule, command, region, restart_policy) VALUES (?, ?, ?, ?, ?, ?);`
	_, err := s.Exec(insertScheduleSQL,
		req.AppName,
		req.Image,
		req.Schedule,
		req.Command,
		req.Region,
		req.RestartPolicy,
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
