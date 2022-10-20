package health

import "context"

// Simple aggregate a list of Checkers
type Simple struct {
	checkers []Checker
}

// NewSimple creates a new Simple
func NewSimple(checker ...Checker) *Simple {
	return &Simple{checkers: checker}
}

// AddChecker add a Checker to the aggregator
func (c *Simple) AddChecker(checker Checker) {
	c.checkers = append(c.checkers, checker)
}

// Check returns the combination of all checkers added
// if some check is not UP, the combined is marked as Down
func (c *Simple) Check(ctx context.Context) ReportDocument {
	return c.check(ctx)
}

func (c *Simple) check(ctx context.Context) ReportDocumentList {
	var res ReportDocumentList

	for _, item := range c.checkers {
		ch := item.Check(ctx)
		res = append(res, ch)
	}

	return res
}
