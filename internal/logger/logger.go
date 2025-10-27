package logger

import (
	"go.uber.org/zap"
)

func New() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return l
}
