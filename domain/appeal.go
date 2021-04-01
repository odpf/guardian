package domain

import "time"

const (
	AppealStatusPending    = "pending"
	AppealStatusActive     = "active"
	AppealStatusRejected   = "rejected"
	AppealStatusTerminated = "terminated"
)

// Appeal struct
type Appeal struct {
	ID            uint                   `json:"id"`
	ResourceID    uint                   `json:"resource_id"`
	PolicyID      string                 `json:"policy_id"`
	PolicyVersion uint                   `json:"policy_version"`
	Status        string                 `json:"status"`
	Email         string                 `json:"email"`
	Labels        map[string]interface{} `json:"labels"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// AppealRepository interface
type AppealRepository interface {
	BulkInsert([]*Appeal) error
}

// AppealService interface
type AppealService interface {
	Create(user string, resourceIDs []uint) ([]*Appeal, error)
}