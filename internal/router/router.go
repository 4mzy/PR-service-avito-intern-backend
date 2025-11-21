package router

import (
	"net/http"
	"pr-reviewer-service/internal/handler"
)

func NewRouter(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/team/add", h.AddTeam)
	mux.HandleFunc("/team/get", h.GetTeam)
	mux.HandleFunc("/users/setIsActive", h.SetIsActive)
	mux.HandleFunc("/users/getReview", h.GetUserReviews)
	mux.HandleFunc("/pullRequest/create", h.CreatePullRequest)
	mux.HandleFunc("/pullRequest/merge", h.MergePullRequest)
	mux.HandleFunc("/pullRequest/reassign", h.ReassignPullRequest)
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/stats", h.GetStatistics)
	mux.HandleFunc("/users/deactivate", h.DeactivateUsers)

	return mux
}

