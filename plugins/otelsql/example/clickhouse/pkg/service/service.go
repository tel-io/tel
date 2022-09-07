package service

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
)

type ClickHouse struct {
	db *sql.DB
}

func New(db *sql.DB) *ClickHouse {
	return &ClickHouse{db: db}
}

func (c *ClickHouse) Setup(ctx context.Context) error {
	if _, err := c.db.ExecContext(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return errors.WithStack(err)
	}

	_, err := c.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			  Col1 UInt8
			, Col2 String
			, Col3 FixedString(3)
			, Col4 UUID
			, Col5 Map(String, UInt8)
			, Col6 Array(String)
			, Col7 Tuple(String, UInt8, Array(Map(String, String)))
			, Col8 DateTime
		) Engine = Memory
	`)

	return errors.WithStack(err)
}

func (c *ClickHouse) Run(ctx context.Context) error {
	if err := c.Setup(ctx); err != nil {
		return errors.WithStack(err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}

		span, ccx := tel.FromCtx(ctx).StartSpan(ctx, "click-batch")
		if err := c.Batch(ccx); err != nil {
			tel.FromCtx(ctx).Panic("batch issue", tel.Error(err))
		}

		span.End()
	}

	return nil
}

func (c *ClickHouse) Batch(ctx context.Context) error {
	scope, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	batch, err := scope.PrepareContext(ctx, "INSERT INTO example (Col1,Col2,Col3,Col4,Col5,Col6,Col7,Col8)")
	if err != nil {
		return errors.WithStack(err)
	}

	for i := 0; i < 20; i++ {
		_, err := batch.ExecContext(ctx,
			uint8(42),
			"ClickHouse",
			"Inc",
			uuid.New(),
			map[string]uint8{"key": 1},             // Map(String, UInt8)
			[]string{"Q", "W", "E", "R", "T", "Y"}, // Array(String)
			[]interface{}{ // Tuple(String, UInt8, Array(Map(String, String)))
				"String Value", uint8(5), []map[string]string{
					map[string]string{"key": "value"},
					map[string]string{"key": "value"},
					map[string]string{"key": "value"},
				},
			},
			time.Now(),
		)

		if err != nil {
			if rollErr := scope.Rollback(); rollErr != nil {
				return errors.WithStack(errors.WithMessage(err, rollErr.Error()))
			}

			return errors.WithStack(err)
		}
	}

	return errors.WithStack(scope.Commit())
}
