package checkers

import (
	health "github.com/d7561985/tel/monitoring/heallth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type grpcClientChecker struct {
	conn *grpc.ClientConn
}

func NewGrpcClientChecker(conn *grpc.ClientConn) health.Checker {
	return &grpcClientChecker{conn: conn}
}

func (ch *grpcClientChecker) Check() health.Health {
	check := health.NewHealth()
	state := ch.conn.GetState()

	check.AddInfo("target", ch.conn.Target())

	if state < connectivity.TransientFailure {
		check.Set(health.UP)
	} else {
		check.Set(health.Down)
	}

	check.AddInfo("state", state.String())

	return check
}
