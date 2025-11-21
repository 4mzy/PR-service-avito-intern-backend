package repository

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"pr-reviewer-service/internal/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateOrUpdate(user *models.User) error {
	query := `INSERT INTO users (user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active`
	_, err := r.db.Exec(query, user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

func (r *UserRepository) GetByID(userID string) (*models.User, error) {
	var user models.User
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) SetIsActive(userID string, isActive bool) error {
	query := `UPDATE users SET is_active = $1 WHERE user_id = $2`
	result, err := r.db.Exec(query, isActive, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) GetActiveUsersByTeam(teamName, excludeUserID string) ([]*models.User, error) {
	query := `SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != $2`
	rows, err := r.db.Query(query, teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func (r *UserRepository) GetUsersByTeam(teamName string) ([]*models.User, error) {
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE team_name = $1`
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func (r *UserRepository) BulkSetIsActive(userIDs []string, isActive bool) error {
	if len(userIDs) == 0 {
		return nil
	}
	query := `UPDATE users SET is_active = $1 WHERE user_id = ANY($2::text[])`
	_, err := r.db.Exec(query, isActive, pq.Array(userIDs))
	return err
}

func (r *UserRepository) GetReviewerCount(userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM pr_reviewers WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (r *UserRepository) GetAuthoredPRCount(userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM pull_requests WHERE author_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (r *UserRepository) GetAllUsersStats() ([]*models.UserStats, error) {
	query := `
		SELECT 
			u.user_id,
			u.username,
			COALESCE(COUNT(DISTINCT pr.user_id), 0) as assigned_count,
			COALESCE(COUNT(DISTINCT p.pull_request_id), 0) as authored_count
		FROM users u
		LEFT JOIN pr_reviewers pr ON u.user_id = pr.user_id
		LEFT JOIN pull_requests p ON u.user_id = p.author_id
		GROUP BY u.user_id, u.username
		ORDER BY u.user_id`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.UserStats
	for rows.Next() {
		var s models.UserStats
		if err := rows.Scan(&s.UserID, &s.Username, &s.AssignedAsReviewerCount, &s.AuthoredPRCount); err != nil {
			return nil, err
		}
		stats = append(stats, &s)
	}
	return stats, rows.Err()
}

