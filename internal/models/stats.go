package models

type UserStats struct {
	UserID                  string `json:"user_id"`
	Username                string `json:"username"`
	AssignedAsReviewerCount int    `json:"assigned_as_reviewer_count"`
	AuthoredPRCount         int    `json:"authored_pr_count"`
}

type DeactivationRequest struct {
	TeamName string   `json:"team_name"`
	UserIDs  []string `json:"user_ids"`
}

type DeactivationResponse struct {
	TeamName            string   `json:"team_name"`
	DeactivatedUsers   []string `json:"deactivated_users"`
	ReassignedPRs       []string `json:"reassigned_prs"`
	FailedReassignments []string `json:"failed_reassignments,omitempty"`
}

