package task

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/openzipkin/zipkin-go"
	escontext "github.com/purini-to/envoy-sample/context"
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
	FindAll(ctx context.Context) ([]*Task, error)
	FindByID(ctx context.Context, id string) (*Task, error)
}

type repository struct {
	db     *sql.DB
	tracer *zipkin.Tracer
}

func (r *repository) FindAll(ctx context.Context) ([]*Task, error) {
	rows, err := r.Query(ctx, `SELECT * FROM task ORDER BY id`)
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

func (r *repository) FindByID(ctx context.Context, id string) (*Task, error) {
	rows, err := r.Query(ctx, `SELECT * FROM task WHERE id = ?`, id)
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

func (r *repository) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	span := r.tracer.StartSpan("query", zipkin.Parent(escontext.GetSpanContext(ctx)))
	span.Tag("query", query)
	span.Tag("args", fmt.Sprintf("%v", args))
	defer span.Finish()
	return r.db.Query(query, args...)
}

func NewRepository(db *sql.DB, tracer *zipkin.Tracer) (Repository, error) {
	return &repository{
		db:     db,
		tracer: tracer,
	}, nil
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
