package service

import (
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) SetIsActive(userID string, isActive bool) (*models.User, error) {
	err := s.userRepo.SetIsActive(userID, isActive)
	if err != nil {
		return nil, err
	}
	return s.userRepo.GetByID(userID)
}

