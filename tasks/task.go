package tasks

type Task interface {
	Starts() error
	Stop() error
}
