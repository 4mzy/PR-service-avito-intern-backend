package service

import (
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
)

type TeamService struct {
	teamRepo *repository.TeamRepository
}

func NewTeamService(teamRepo *repository.TeamRepository) *TeamService {
	return &TeamService{teamRepo: teamRepo}
}

func (s *TeamService) CreateTeam(team *models.Team) error {
	return s.teamRepo.Create(team)
}

func (s *TeamService) GetTeam(teamName string) (*models.Team, error) {
	return s.teamRepo.GetByName(teamName)
}

