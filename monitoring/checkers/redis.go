package checkers

import (
	health "github.com/d7561985/tel/v2/monitoring/heallth"
)

// Redis is a interface used to abstract the access of the Version string
type Redis interface {
	GetVersion() (string, error)
}

// Checker is a checker that check a given redis
type Checker struct {
	Redis Redis
}

// NewCheckerWithRedis returns a new redis.Checker configured with a custom Redis implementation
func NewCheckerWithRedis(redis Redis) Checker {
	return Checker{Redis: redis}
}

// Check obtain the version string from redis info command
func (c Checker) Check() health.Health {
	h := health.NewHealth()

	version, err := c.Redis.GetVersion()
	if err != nil {
		h.Set(health.Down)
		h.AddInfo("error", err.Error())

		return h
	}

	h.Set(health.UP)
	h.AddInfo("version", version)

	return h
}
