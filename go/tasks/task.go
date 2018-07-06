package tasks

type Task interface {
	GetStatus() Status
	Starts() error
	Stop() error
}

type Status string

const (
	Pending Status = "pending"
	Running Status = "running"
	Stopped Status = "stopped"
)
