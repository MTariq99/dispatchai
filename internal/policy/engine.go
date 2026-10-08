package policy

import "github.com/mtariq99/dispatchai/models"

// Engine runs every registered rule against a call, in order. First
// non-Allow outcome wins — Deny and RequireApproval short-circuit the
// remaining rules, since there's no reason to keep evaluating once one
// rule has already decided the call shouldn't proceed as-is.
type Engine struct {
	rules []Rule
}

func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: rules}
}

func (e *Engine) Evaluate(call *models.Call, execCtx *models.ExecutionContext, callCounts map[string]int) Decision {
	for _, rule := range e.rules {
		decision := rule(call, execCtx, callCounts)
		if decision.Outcome != Allow {
			return decision
		}
	}
	return AllowDecision()
}
