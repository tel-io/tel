package checkers

import (
	health "github.com/d7561985/tel/monitoring/heallth"
)

type ShutDownChecker struct{}

func ShutDown() *ShutDownChecker {
	return &ShutDownChecker{}
}

func (ShutDownChecker) Check() health.Health {
	check := health.NewHealth()
	check.Set(health.Down)

	return check
}
