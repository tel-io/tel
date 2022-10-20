package health

import (
	"context"
)

// Checker is interface used to provide an indication of application health.
type Checker interface {
	Check(context.Context) ReportDocument
}

// CheckerFunc is an adapter to allow the use of
// ordinary go functions as Checkers.
type CheckerFunc func(ctx context.Context) ReportDocument

func (f CheckerFunc) Check(ctx context.Context) ReportDocument {
	return f(ctx)
}

// Controller of checker set
type Controller interface {
	AddChecker(checker Checker)

	Checker
}
