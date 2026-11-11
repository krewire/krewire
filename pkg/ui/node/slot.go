package node

import (
	"bytes"
	"context"
	"html/template"
	"io"
)

// slotContextKey is the private context key for slot inheritance.
type slotContextKey struct{}

// SlotMap associates named slots with their corresponding child nodes.
type SlotMap map[string]Node

// ContextWithSlots attaches a SlotMap to the context, merging with any existing slots.
func ContextWithSlots(ctx context.Context, slots SlotMap) context.Context {
	if len(slots) == 0 {
		return ctx
	}
	existing := SlotsFromContext(ctx)
	merged := make(SlotMap, len(existing)+len(slots))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range slots {
		merged[k] = v
	}
	return context.WithValue(ctx, slotContextKey{}, merged)
}

// SlotsFromContext retrieves the current SlotMap from context.
func SlotsFromContext(ctx context.Context) SlotMap {
	if ctx == nil {
		return nil
	}
	if sm, ok := ctx.Value(slotContextKey{}).(SlotMap); ok {
		return sm
	}
	return nil
}

// slotNode renders either a named slot from context or its fallback nodes.
type slotNode struct {
	name     string
	fallback []Node
}

// Slot creates a node outlet for a named slot.
// If the slot is provided in the parent context, that slot's node is rendered.
// Otherwise, the fallback nodes (if any) are rendered.
func Slot(name string, fallback ...Node) Node {
	filtered := make([]Node, 0, len(fallback))
	for _, f := range fallback {
		if f != nil {
			filtered = append(filtered, f)
		}
	}
	return &slotNode{
		name:     name,
		fallback: filtered,
	}
}

func (s *slotNode) Render(ctx context.Context, w io.Writer) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	slots := SlotsFromContext(ctx)
	if slots != nil {
		if node, exists := slots[s.name]; exists && node != nil {
			return node.Render(ctx, w)
		}
	}
	for _, fb := range s.fallback {
		if fb != nil {
			if err := fb.Render(ctx, w); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *slotNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	var buf bytes.Buffer
	if err := s.Render(ctx, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// slotProviderNode injects a SlotMap into the rendering context of child.
type slotProviderNode struct {
	slots SlotMap
	child Node
}

// WithSlots wraps a node with a SlotMap that will be available
// to any Slot outlets rendered within its subtree.
func WithSlots(child Node, slots SlotMap) Node {
	if child == nil {
		return nil
	}
	return &slotProviderNode{
		slots: slots,
		child: child,
	}
}

func (sp *slotProviderNode) Render(ctx context.Context, w io.Writer) error {
	subCtx := ContextWithSlots(ctx, sp.slots)
	return sp.child.Render(subCtx, w)
}

func (sp *slotProviderNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	var buf bytes.Buffer
	if err := sp.Render(ctx, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
