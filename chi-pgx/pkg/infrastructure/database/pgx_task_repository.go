package database

import (
	"chi-pgx/pkg/config"
	"chi-pgx/pkg/utils/domain"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxTaskRepository struct {
	db *pgxpool.Pool
}

var (
	once       sync.Once
	repository *PgxTaskRepository
)

// NewPgxTaskRepository initializes the connection using pgx
func NewPgxTaskRepository(cfg config.Config) (domain.TaskRepository, error) {
	var err error
	databaseUrl := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	once.Do(func() {
		db, dbErr := pgxpool.New(context.Background(), databaseUrl)
		if dbErr != nil {
			err = dbErr
			log.Fatalf("Database Connection Error: %v", err)
			return
		}

		if pingErr := db.Ping(context.Background()); pingErr != nil {
			err = pingErr
			log.Fatalf("Database Ping Error: %v", err)
			return
		}

		repository = &PgxTaskRepository{db: db}
	})

	return repository, err
}

// GetTaskByID fetches a task by its ID using pgx
func (r *PgxTaskRepository) GetTaskByID(id int64) (*domain.Task, error) {
	task := &domain.Task{}
	query := "SELECT id, title, description FROM tasks WHERE id = $1"
	row := r.db.QueryRow(context.Background(), query, id)

	err := row.Scan(&task.ID, &task.Title, &task.Description)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// CreateTask creates a new task using pgx
func (r *PgxTaskRepository) CreateTask(task *domain.Task) error {
	query := "INSERT INTO tasks (title, description) VALUES ($1, $2)"
	_, err := r.db.Exec(context.Background(), query, task.Title, task.Description)
	return err
}
