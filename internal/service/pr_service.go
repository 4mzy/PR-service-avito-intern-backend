package service

import (
	"errors"
	"math/rand"
	"time"

	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
)

var (
	ErrAuthorNotFound    = errors.New("author not found")
	ErrTeamNotFound      = errors.New("team not found")
	ErrPRMerged          = errors.New("PR is already merged")
	ErrReviewerNotAssigned = errors.New("reviewer is not assigned")
	ErrNoCandidate       = errors.New("no active replacement candidate")
)

type PullRequestService struct {
	prRepo     *repository.PullRequestRepository
	userRepo   *repository.UserRepository
	teamRepo   *repository.TeamRepository
	randSource *rand.Rand
}

func NewPullRequestService(
	prRepo *repository.PullRequestRepository,
	userRepo *repository.UserRepository,
	teamRepo *repository.TeamRepository,
) *PullRequestService {
	return &PullRequestService{
		prRepo:     prRepo,
		userRepo:   userRepo,
		teamRepo:   teamRepo,
		randSource: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *PullRequestService) CreatePR(prID, prName, authorID string) (*models.PullRequest, error) {
	author, err := s.userRepo.GetByID(authorID)
	if err != nil {
		return nil, ErrAuthorNotFound
	}

	candidates, err := s.userRepo.GetActiveUsersByTeam(author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	var reviewers []string
	maxReviewers := 2
	if len(candidates) < maxReviewers {
		maxReviewers = len(candidates)
	}

	if maxReviewers > 0 {
		shuffled := make([]*models.User, len(candidates))
		copy(shuffled, candidates)
		s.shuffleUsers(shuffled)
		
		for i := 0; i < maxReviewers; i++ {
			reviewers = append(reviewers, shuffled[i].UserID)
		}
	}

	pr := &models.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            models.StatusOpen,
		AssignedReviewers: reviewers,
	}

	err = s.prRepo.Create(pr)
	if err != nil {
		return nil, err
	}

	return s.prRepo.GetByID(prID)
}

func (s *PullRequestService) MergePR(prID string) (*models.PullRequest, error) {
	err := s.prRepo.Merge(prID)
	if err != nil {
		return nil, err
	}
	return s.prRepo.GetByID(prID)
}

func (s *PullRequestService) ReassignReviewer(prID, oldUserID string) (*models.PullRequest, string, error) {
	pr, err := s.prRepo.GetByID(prID)
	if err != nil {
		return nil, "", err
	}

	if pr.Status == models.StatusMerged {
		return nil, "", ErrPRMerged
	}

	assigned := false
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == oldUserID {
			assigned = true
			break
		}
	}
	if !assigned {
		return nil, "", ErrReviewerNotAssigned
	}

	oldReviewer, err := s.userRepo.GetByID(oldUserID)
	if err != nil {
		return nil, "", err
	}

	candidates, err := s.userRepo.GetActiveUsersByTeam(oldReviewer.TeamName, oldUserID)
	if err != nil {
		return nil, "", err
	}

	filtered := make([]*models.User, 0)
	for _, candidate := range candidates {
		if candidate.UserID != pr.AuthorID {
			filtered = append(filtered, candidate)
		}
	}
	candidates = filtered

	filtered = make([]*models.User, 0)
	for _, candidate := range candidates {
		isAlreadyAssigned := false
		for _, assignedReviewerID := range pr.AssignedReviewers {
			if candidate.UserID == assignedReviewerID && candidate.UserID != oldUserID {
				isAlreadyAssigned = true
				break
			}
		}
		if !isAlreadyAssigned {
			filtered = append(filtered, candidate)
		}
	}
	candidates = filtered

	if len(candidates) == 0 {
		return nil, "", ErrNoCandidate
	}

	selected := candidates[s.randSource.Intn(len(candidates))]

	err = s.prRepo.ReassignReviewer(prID, oldUserID, selected.UserID)
	if err != nil {
		return nil, "", err
	}

	updatedPR, err := s.prRepo.GetByID(prID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, selected.UserID, nil
}

func (s *PullRequestService) GetPRsByReviewer(userID string) ([]*models.PullRequestShort, error) {
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	return s.prRepo.GetPRsByReviewer(userID)
}

func (s *PullRequestService) shuffleUsers(users []*models.User) {
	for i := len(users) - 1; i > 0; i-- {
		j := s.randSource.Intn(i + 1)
		users[i], users[j] = users[j], users[i]
	}
}

