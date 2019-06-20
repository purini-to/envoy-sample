package task

import (
	"database/sql"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Repository interface {
	FindAll() ([]*Task, error)
	FindByID(id string) (*Task, error)
}

type repository struct {
	db *sql.DB
}

func (r *repository) FindAll() ([]*Task, error) {
	rows, err := r.db.Query(`SELECT * FROM task ORDER BY id`)
	if err != nil {
		return []*Task{}, err
	}
	defer rows.Close()

	tasks, err := scanBindTask(rows)
	if err != nil {
		return []*Task{}, err
	}

	return tasks, nil
}

func (r *repository) FindByID(id string) (*Task, error) {
	rows, err := r.db.Query(`SELECT * FROM task WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks, err := scanBindTask(rows)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, nil
	}

	return tasks[0], nil
}

func NewRepository(db *sql.DB) Repository {
	return &repository{
		db: db,
	}
}

func scanBindTask(rows *sql.Rows) ([]*Task, error) {
	var tasks []*Task
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Title, &task.State, &task.CreatedAt, &task.UpdatedAt)

		if err != nil {
			return []*Task{}, err
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}
