package entity

type Status string

const (
	StatusRunning Status = "RUNNING"
	StatusStopped Status = "STOPPED"
	StatusError   Status = "ERROR"
)
