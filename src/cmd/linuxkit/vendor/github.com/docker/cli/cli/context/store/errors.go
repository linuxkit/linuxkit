package store

import cerrdefs "github.com/containerd/errdefs"

func invalidParameter(err error) error {
	if err == nil || cerrdefs.IsInvalidArgument(err) {
		return err
	}
	return invalidParameterErr{err}
}

type invalidParameterErr struct{ error }

func (invalidParameterErr) InvalidParameter() {}

func notFound(err error) error {
	if err == nil || cerrdefs.IsNotFound(err) {
		return err
	}
	return notFoundErr{err}
}

type notFoundErr struct{ error }

func (notFoundErr) NotFound() {}
func (e notFoundErr) Unwrap() error {
	return e.error
}
