package errors

import "errors"

// ErrNoClue is for the situation that existing information is not enough to make a decision. For example, Router may return this error when there is no suitable route.
var ErrNoClue = errors.New("not enough information for making a decision")

var ErrCombine = errors.New("combine error")
