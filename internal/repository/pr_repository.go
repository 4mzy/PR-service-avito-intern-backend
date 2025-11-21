package repository

import (
	"database/sql"
	"errors"
	"time"

	"pr-reviewer-service/internal/models"
)

var (
	ErrPRExists  = errors.New("PR already exists")
	ErrPRNotFound = errors.New("PR not found")
)

type PullRequestRepository struct {
	db *sql.DB
}

func NewPullRequestRepository(db *sql.DB) *PullRequestRepository {
	return &PullRequestRepository{db: db}
}

func (r *PullRequestRepository) Create(pr *models.PullRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", pr.PullRequestID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrPRExists
	}

	now := time.Now()
	_, err = tx.Exec(
		`INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, now)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.AssignedReviewers {
		_, err = tx.Exec(
			`INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`,
			pr.PullRequestID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PullRequestRepository) GetByID(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var createdAt, mergedAt sql.NullTime

	query := `SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests WHERE pull_request_id = $1`
	err := r.db.QueryRow(query, prID).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)
	if err == sql.ErrNoRows {
		return nil, ErrPRNotFound
	}
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	reviewersQuery := `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1`
	rows, err := r.db.Query(reviewersQuery, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	return &pr, rows.Err()
}

func (r *PullRequestRepository) Merge(prID string) error {
	query := `UPDATE pull_requests 
		SET status = 'MERGED', merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1`
	result, err := r.db.Exec(query, prID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrPRNotFound
	}
	return nil
}

func (r *PullRequestRepository) ReassignReviewer(prID, oldUserID, newUserID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRow("SELECT status FROM pull_requests WHERE pull_request_id = $1", prID).Scan(&status)
	if err == sql.ErrNoRows {
		return ErrPRNotFound
	}
	if err != nil {
		return err
	}

	var exists bool
	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2)",
		prID, oldUserID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("reviewer is not assigned")
	}

	_, err = tx.Exec(
		`UPDATE pr_reviewers SET user_id = $1 WHERE pull_request_id = $2 AND user_id = $3`,
		newUserID, prID, oldUserID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PullRequestRepository) GetPRsByReviewer(userID string) ([]*models.PullRequestShort, error) {
	query := `
		SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status
		FROM pull_requests p
		INNER JOIN pr_reviewers pr ON p.pull_request_id = pr.pull_request_id
		WHERE pr.user_id = $1
		ORDER BY p.created_at DESC`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, &pr)
	}
	return prs, rows.Err()
}

func (r *PullRequestRepository) GetOpenPRsWithReviewer(userID string) ([]*models.PullRequest, error) {
	query := `
		SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status, p.created_at, p.merged_at
		FROM pull_requests p
		INNER JOIN pr_reviewers pr ON p.pull_request_id = pr.pull_request_id
		WHERE pr.user_id = $1 AND p.status = 'OPEN'`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*models.PullRequest
	for rows.Next() {
		var pr models.PullRequest
		var createdAt, mergedAt sql.NullTime
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			pr.CreatedAt = &createdAt.Time
		}
		if mergedAt.Valid {
			pr.MergedAt = &mergedAt.Time
		}

		reviewersQuery := `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1`
		reviewerRows, err := r.db.Query(reviewersQuery, pr.PullRequestID)
		if err != nil {
			return nil, err
		}
		for reviewerRows.Next() {
			var reviewerID string
			if err := reviewerRows.Scan(&reviewerID); err != nil {
				reviewerRows.Close()
				return nil, err
			}
			pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
		}
		reviewerRows.Close()

		prs = append(prs, &pr)
	}
	return prs, rows.Err()
}

