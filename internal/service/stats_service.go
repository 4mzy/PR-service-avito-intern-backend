package service

import (
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
)

type StatsService struct {
	userRepo *repository.UserRepository
}

func NewStatsService(userRepo *repository.UserRepository) *StatsService {
	return &StatsService{userRepo: userRepo}
}

func (s *StatsService) GetStatistics() ([]*models.UserStats, error) {
	return s.userRepo.GetAllUsersStats()
}

