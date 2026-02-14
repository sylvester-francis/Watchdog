package domain

import "errors"

var (
	ErrAgentLimitReached   = errors.New("agent limit reached for current plan")
	ErrMonitorLimitReached = errors.New("monitor limit reached for current plan")
)
