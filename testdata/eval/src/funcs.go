package sample

import "context"

type Service struct {
	name string
}

func (s *Service) Run(ctx context.Context, job string) error {
	_ = job
	return nil
}

func (s *Service) Describe() string {
	return s.name
}

func Execute(job string) error {
	_ = job
	return nil
}

func NewService(name string) *Service {
	return &Service{name: name}
}

func Transform[T any](ctx context.Context, in []T) []T {
	_ = ctx
	return in
}
