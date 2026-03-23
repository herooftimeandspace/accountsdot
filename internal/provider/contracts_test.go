package provider_test

import (
	"errors"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want provider.ErrorClass
	}{
		{name: "transient", err: provider.ProviderError{Class: provider.ErrorClassTransient, Err: errors.New("retry")}, want: provider.ErrorClassTransient},
		{name: "blocked", err: provider.ProviderError{Class: provider.ErrorClassBlocked, Err: errors.New("blocked")}, want: provider.ErrorClassBlocked},
		{name: "manual", err: provider.ProviderError{Class: provider.ErrorClassManual, Err: errors.New("manual")}, want: provider.ErrorClassManual},
		{name: "fatal default", err: errors.New("boom"), want: provider.ErrorClassFatal},
	}
	for _, tc := range tests {
		if got := provider.ClassifyError(tc.err); got != tc.want {
			t.Fatalf("%s: ClassifyError() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestProviderErrorErrorAndUnwrap(t *testing.T) {
	err := errors.New("inner")
	providerErr := provider.ProviderError{Class: provider.ErrorClassManual, Err: err}

	if providerErr.Error() != "inner" {
		t.Fatalf("expected wrapped error text, got %q", providerErr.Error())
	}
	if !errors.Is(providerErr, err) {
		t.Fatal("expected ProviderError to unwrap to the inner error")
	}

	classOnly := provider.ProviderError{Class: provider.ErrorClassBlocked}
	if classOnly.Error() != "blocked" {
		t.Fatalf("expected class-only error text, got %q", classOnly.Error())
	}
}
