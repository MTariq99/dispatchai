package run

type State string

const (
	StateCreated            State = "created"
	StateRunning            State = "running"
	StateWaitingForTools    State = "waiting_for_tools"
	StateWaitingForApproval State = "waiting_for_approval"
	StateCompleted          State = "completed"
	StateFailed             State = "failed"
	StateCancelled          State = "cancelled"
)

func (s State) IsTerminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateCancelled:
		return true
	default:
		return false
	}
}

func (s State) CanTransitionTo(next State) bool {
	switch s {

	case StateCreated:
		return next == StateRunning || next == StateCancelled

	case StateRunning:
		return next == StateWaitingForTools || next == StateWaitingForApproval || next == StateCompleted || next == StateFailed || next == StateCancelled

	case StateWaitingForTools:
		return next == StateRunning || next == StateFailed || next == StateCancelled

	case StateWaitingForApproval:
		return next == StateRunning || next == StateFailed || next == StateCancelled

	case StateCompleted, StateFailed, StateCancelled:
		return false

	default:
		return false
	}
}
