package fakeengine_test

import (
	"testing"

	"github.com/krewire/krewire/toolchain/adapter"
	"github.com/krewire/krewire/toolchain/conformance"
	"github.com/krewire/krewire/toolchain/fakeengine"
)

func newEngine(*testing.T) adapter.Engine { return &fakeengine.Engine{} }

// TestReferenceEngineConforms runs the conformance suite in-process.
func TestReferenceEngineConforms(t *testing.T) { conformance.Run(t, newEngine) }

// TestReferenceEngineConformsOverStdio runs it across the JSON-RPC boundary.
func TestReferenceEngineConformsOverStdio(t *testing.T) { conformance.RunStdio(t, newEngine) }
