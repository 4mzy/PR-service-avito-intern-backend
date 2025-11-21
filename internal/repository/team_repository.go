package repository

import (
	"database/sql"
	"errors"

	"pr-reviewer-service/internal/models"
)

var (
	ErrTeamExists  = errors.New("team already exists")
	ErrTeamNotFound = errors.New("team not found")
)

type TeamRepository struct {
	db       *sql.DB
	userRepo *UserRepository
}

func NewTeamRepository(db *sql.DB, userRepo *UserRepository) *TeamRepository {
	return &TeamRepository{
		db:       db,
		userRepo: userRepo,
	}
}

func (r *TeamRepository) Create(team *models.Team) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", team.TeamName).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrTeamExists
	}

	_, err = tx.Exec("INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
	if err != nil {
		return err
	}

	for _, member := range team.Members {
		if member.UserID == "" {
			continue
		}
		user := &models.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive:  member.IsActive,
		}
		query := `INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET
				username = EXCLUDED.username,
				team_name = EXCLUDED.team_name,
				is_active = EXCLUDED.is_active`
		_, err = tx.Exec(query, user.UserID, user.Username, user.TeamName, user.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TeamRepository) GetByName(teamName string) (*models.Team, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTeamNotFound
	}

	users, err := r.userRepo.GetUsersByTeam(teamName)
	if err != nil {
		return nil, err
	}

	members := make([]models.TeamMember, len(users))
	for i, user := range users {
		members[i] = models.TeamMember{
			UserID:   user.UserID,
			Username: user.Username,
			IsActive: user.IsActive,
		}
	}

	return &models.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

