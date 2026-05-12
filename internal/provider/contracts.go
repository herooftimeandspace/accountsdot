package provider

import (
	"context"
	"errors"
)

type ErrorClass string

const (
	ErrorClassTransient ErrorClass = "transient"
	ErrorClassBlocked   ErrorClass = "blocked"
	ErrorClassManual    ErrorClass = "manual"
	ErrorClassFatal     ErrorClass = "fatal"
)

type ProviderReference struct {
	ID string
}

type ProviderSnapshot map[string]any

type ProviderIntent struct {
	Operation string
	Payload   map[string]any
}

type ApplyResult struct {
	Applied   bool
	Reference ProviderReference
}

type PublishSpec struct {
	WorkbookID string
}

type StageResult struct {
	WorkbookID string
	StageID    string
}

type ValidationResult struct {
	Valid bool
}

type PointerSpec struct {
	WorkbookID string
	StageID    string
}

type SyncEvaluation struct {
	Matched bool
	Payload map[string]any
	Issues  []string
}

type WorkflowProvider interface {
	ReadExisting(context.Context, ProviderReference) (ProviderSnapshot, error)
	ApplyIntent(context.Context, ProviderIntent) (ApplyResult, error)
}

type SyncProvider interface {
	ReadExisting(context.Context, ProviderReference) (ProviderSnapshot, error)
	Evaluate(context.Context, ProviderReference) (SyncEvaluation, error)
}

type SheetPublisher interface {
	StageWorkbook(context.Context, PublishSpec) (StageResult, error)
	ValidateSentinel(context.Context, StageResult) (ValidationResult, error)
	ApplyPointers(context.Context, PointerSpec) (ApplyResult, error)
}

type ProviderError struct {
	Class ErrorClass
	Err   error
}

func (e ProviderError) Error() string {
	if e.Err == nil {
		return string(e.Class)
	}
	return e.Err.Error()
}

func (e ProviderError) Unwrap() error {
	return e.Err
}

func ClassifyError(err error) ErrorClass {
	var providerErr ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Class
	}
	return ErrorClassFatal
}
