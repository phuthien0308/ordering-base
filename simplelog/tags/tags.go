package tags

import "go.uber.org/zap"

type T = zap.Field

var (
	String   = zap.String
	Int      = zap.Int
	Bool     = zap.Bool
	Any      = zap.Any
	Error    = zap.Error
	Int64    = zap.Int64
	Uint64   = zap.Uint64
	Float64  = zap.Float64
	Duration = zap.Duration
	Time     = zap.Time
)
