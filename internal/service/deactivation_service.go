package service

import (
	"fmt"
	"time"

	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
)

type DeactivationService struct {
	userRepo  *repository.UserRepository
	prRepo    *repository.PullRequestRepository
	prService *PullRequestService
}

func NewDeactivationService(
	userRepo *repository.UserRepository,
	prRepo *repository.PullRequestRepository,
	prService *PullRequestService,
) *DeactivationService {
	return &DeactivationService{
		userRepo:  userRepo,
		prRepo:    prRepo,
		prService: prService,
	}
}

func (s *DeactivationService) DeactivateUsers(teamName string, userIDs []string) (*models.DeactivationResponse, error) {
	startTime := time.Now()

	_, err := s.userRepo.GetUsersByTeam(teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	teamUsers, err := s.userRepo.GetUsersByTeam(teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team users: %w", err)
	}

	userMap := make(map[string]bool)
	for _, u := range teamUsers {
		userMap[u.UserID] = true
	}

	validUserIDs := make([]string, 0)
	for _, userID := range userIDs {
		if userMap[userID] {
			validUserIDs = append(validUserIDs, userID)
		}
	}

	if len(validUserIDs) == 0 {
		return &models.DeactivationResponse{
			TeamName:         teamName,
			DeactivatedUsers: []string{},
			ReassignedPRs:    []string{},
		}, nil
	}

	err = s.userRepo.BulkSetIsActive(validUserIDs, false)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate users: %w", err)
	}

	reassignedPRs := make([]string, 0)
	failedReassignments := make([]string, 0)

	for _, userID := range validUserIDs {
		openPRs, err := s.prRepo.GetOpenPRsWithReviewer(userID)
		if err != nil {
			continue
		}

		for _, pr := range openPRs {
			if time.Since(startTime) > 90*time.Millisecond {
				failedReassignments = append(failedReassignments, pr.PullRequestID)
				continue
			}

			_, _, err := s.prService.ReassignReviewer(pr.PullRequestID, userID)
			if err != nil {
				failedReassignments = append(failedReassignments, pr.PullRequestID)
			} else {
				reassignedPRs = append(reassignedPRs, pr.PullRequestID)
			}
		}
	}

	return &models.DeactivationResponse{
		TeamName:            teamName,
		DeactivatedUsers:    validUserIDs,
		ReassignedPRs:       reassignedPRs,
		FailedReassignments: failedReassignments,
	}, nil
}

