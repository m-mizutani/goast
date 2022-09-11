package goast

type FailCase failCase
type EvalOutput struct {
	Fail []*FailCase `json:"fail"`
}
