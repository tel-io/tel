package checkers

import (
	"net/http"
	"time"

	health "github.com/d7561985/tel/monitoring/heallth"
)

type DomainChecker struct {
	URL     string
	Timeout time.Duration
}

func NewDomain(url string) DomainChecker {
	return DomainChecker{URL: url, Timeout: 5 * time.Second}
}

func NewDomainWithTimeout(url string, timeout time.Duration) DomainChecker {
	return DomainChecker{URL: url, Timeout: timeout}
}

func (u DomainChecker) Check() health.Health {
	client := http.Client{
		Timeout: u.Timeout,
	}

	check := health.NewHealth()

	//nolint: noctx
	resp, err := client.Head(u.URL)

	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		check.Set(health.Down)
		check.AddInfo("code", http.StatusBadRequest)

		return check
	}

	if resp.StatusCode < http.StatusInternalServerError {
		check.Set(health.UP)
	} else {
		check.Set(health.Down)
	}

	check.AddInfo("code", resp.StatusCode)

	return check
}
