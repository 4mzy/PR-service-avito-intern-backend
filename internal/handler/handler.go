package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"
)

type Handler struct {
	teamService         *service.TeamService
	userService         *service.UserService
	prService           *service.PullRequestService
	statsService        *service.StatsService
	deactivationService *service.DeactivationService
}

func NewHandler(
	teamService *service.TeamService,
	userService *service.UserService,
	prService *service.PullRequestService,
	statsService *service.StatsService,
	deactivationService *service.DeactivationService,
) *Handler {
	return &Handler{
		teamService:         teamService,
		userService:         userService,
		prService:           prService,
		statsService:        statsService,
		deactivationService: deactivationService,
	}
}

const (
	ErrorCodeTeamExists   = "TEAM_EXISTS"
	ErrorCodePRExists     = "PR_EXISTS"
	ErrorCodePRMerged     = "PR_MERGED"
	ErrorCodeNotAssigned  = "NOT_ASSIGNED"
	ErrorCodeNoCandidate  = "NO_CANDIDATE"
	ErrorCodeNotFound     = "NOT_FOUND"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) writeError(w http.ResponseWriter, code string, message string, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func (h *Handler) checkMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

type AddTeamRequest struct {
	TeamName string                `json:"team_name"`
	Members  []models.TeamMember   `json:"members"`
}

func (h *Handler) AddTeam(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodPost) {
		return
	}
	
	var req AddTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, ErrorCodeNotFound, "invalid request body", http.StatusBadRequest)
		return
	}
	
	if len(req.Members) == 0 {
		h.writeError(w, ErrorCodeNotFound, "no members provided", http.StatusBadRequest)
		return
	}
	
	for _, m := range req.Members {
		if m.UserID == "" {
			h.writeError(w, ErrorCodeNotFound, fmt.Sprintf("member with empty user_id found: username=%q", m.Username), http.StatusBadRequest)
			return
		}
	}

	team := &models.Team{
		TeamName: req.TeamName,
		Members:  req.Members,
	}

	err := h.teamService.CreateTeam(team)
	if err != nil {
		if err == repository.ErrTeamExists {
			h.writeError(w, ErrorCodeTeamExists, "team_name already exists", http.StatusBadRequest)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	createdTeam, err := h.teamService.GetTeam(req.TeamName)
	if err != nil {
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"team": createdTeam,
	})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodGet) {
		return
	}
	
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.writeError(w, ErrorCodeNotFound, "team_name is required", http.StatusBadRequest)
		return
	}

	team, err := h.teamService.GetTeam(teamName)
	if err != nil {
		if err == repository.ErrTeamNotFound {
			h.writeError(w, ErrorCodeNotFound, "team not found", http.StatusNotFound)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

func (h *Handler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodPost) {
		return
	}
	
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, ErrorCodeNotFound, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userService.SetIsActive(req.UserID, req.IsActive)
	if err != nil {
		if err == repository.ErrUserNotFound {
			h.writeError(w, ErrorCodeNotFound, "user not found", http.StatusNotFound)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

func (h *Handler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodPost) {
		return
	}
	
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, ErrorCodeNotFound, "invalid request body", http.StatusBadRequest)
		return
	}

	pr, err := h.prService.CreatePR(req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		if err == service.ErrAuthorNotFound || err == service.ErrTeamNotFound {
			h.writeError(w, ErrorCodeNotFound, "author or team not found", http.StatusNotFound)
			return
		}
		if err == repository.ErrPRExists {
			h.writeError(w, ErrorCodePRExists, "PR id already exists", http.StatusConflict)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodPost) {
		return
	}
	
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, ErrorCodeNotFound, "invalid request body", http.StatusBadRequest)
		return
	}

	pr, err := h.prService.MergePR(req.PullRequestID)
	if err != nil {
		if err == repository.ErrPRNotFound {
			h.writeError(w, ErrorCodeNotFound, "PR not found", http.StatusNotFound)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) ReassignPullRequest(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodPost) {
		return
	}
	
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, ErrorCodeNotFound, "invalid request body", http.StatusBadRequest)
		return
	}

	pr, newUserID, err := h.prService.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		if err == repository.ErrPRNotFound || err == repository.ErrUserNotFound {
			h.writeError(w, ErrorCodeNotFound, "PR or user not found", http.StatusNotFound)
			return
		}
		if err == service.ErrPRMerged {
			h.writeError(w, ErrorCodePRMerged, "cannot reassign on merged PR", http.StatusConflict)
			return
		}
		if err == service.ErrReviewerNotAssigned {
			h.writeError(w, ErrorCodeNotAssigned, "reviewer is not assigned to this PR", http.StatusConflict)
			return
		}
		if err == service.ErrNoCandidate {
			h.writeError(w, ErrorCodeNoCandidate, "no active replacement candidate in team", http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "reviewer is not assigned") {
			h.writeError(w, ErrorCodeNotAssigned, "reviewer is not assigned to this PR", http.StatusConflict)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr":          pr,
		"replaced_by": newUserID,
	})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodGet) {
		return
	}
	
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.writeError(w, ErrorCodeNotFound, "user_id is required", http.StatusBadRequest)
		return
	}

	prs, err := h.prService.GetPRsByReviewer(userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			h.writeError(w, ErrorCodeNotFound, "user not found", http.StatusNotFound)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":      userID,
		"pull_requests": prs,
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodGet) {
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodGet) {
		return
	}
	
	stats, err := h.statsService.GetStatistics()
	if err != nil {
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"statistics": stats,
	})
}

func (h *Handler) DeactivateUsers(w http.ResponseWriter, r *http.Request) {
	if !h.checkMethod(w, r, http.MethodPost) {
		return
	}
	
	var req models.DeactivationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, ErrorCodeNotFound, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TeamName == "" {
		h.writeError(w, ErrorCodeNotFound, "team_name is required", http.StatusBadRequest)
		return
	}

	response, err := h.deactivationService.DeactivateUsers(req.TeamName, req.UserIDs)
	if err != nil {
		if err == repository.ErrTeamNotFound {
			h.writeError(w, ErrorCodeNotFound, "team not found", http.StatusNotFound)
			return
		}
		h.writeError(w, ErrorCodeNotFound, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

