package errors

type multiErr struct {
	base   error
	causes []error
}

func (e *multiErr) Unwrap() error {
	l := len(e.causes)
	if l > 0 {
		return e.causes[l-1]
	}
	return nil
}

func (e *multiErr) Error() string {
	return e.base.Error()
}

func Multi(base error, causes []error) error {
	return &multiErr{
		base:   base,
		causes: causes,
	}
}
