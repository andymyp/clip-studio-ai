package model

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

type ClipStatus string

const (
	ClipStatusCandidate ClipStatus = "candidate"
	ClipStatusApproved  ClipStatus = "approved"
	ClipStatusRejected  ClipStatus = "rejected"
	ClipStatusRendered  ClipStatus = "rendered"
)
