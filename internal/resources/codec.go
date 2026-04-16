package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// KindCodec owns the typed encode/decode boundary for one resource kind.
type KindCodec[T any] interface {
	Kind() ResourceKind
	DecodeAndValidate(ctx context.Context, scope ResourceScope, raw []byte) (T, error)
	Encode(spec T) ([]byte, error)
	MaxBytes() int
}

// SpecValidator enforces typed invariants after decoding and before persistence.
type SpecValidator[T any] func(ctx context.Context, scope ResourceScope, spec T) (T, error)

type jsonCodec[T any] struct {
	kind      ResourceKind
	maxBytes  int
	validator SpecValidator[T]
}

// NewJSONCodec builds a JSON-backed codec with a typed validation hook.
func NewJSONCodec[T any](kind ResourceKind, maxBytes int, validator SpecValidator[T]) (KindCodec[T], error) {
	normalizedKind := kind.Normalize()
	if err := normalizedKind.Validate("codec.kind"); err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("%w: codec.max_bytes must be positive: %d", ErrValidation, maxBytes)
	}

	return &jsonCodec[T]{
		kind:      normalizedKind,
		maxBytes:  maxBytes,
		validator: validator,
	}, nil
}

func (c *jsonCodec[T]) Kind() ResourceKind {
	return c.kind
}

func (c *jsonCodec[T]) MaxBytes() int {
	return c.maxBytes
}

func (c *jsonCodec[T]) Encode(spec T) ([]byte, error) {
	encoded, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("resources: encode %q spec: %w", c.kind, err)
	}
	if err := validateCodecPayloadSize(len(encoded), c.maxBytes, c.kind, "encode"); err != nil {
		return nil, err
	}
	return append([]byte(nil), encoded...), nil
}

func (c *jsonCodec[T]) DecodeAndValidate(ctx context.Context, scope ResourceScope, raw []byte) (T, error) {
	var zero T
	if ctx == nil {
		return zero, errors.New("resources: codec decode context is required")
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return zero, fmt.Errorf("%w: codec payload is required for kind %q", ErrValidation, c.kind)
	}
	if err := validateCodecPayloadSize(len(trimmed), c.maxBytes, c.kind, "decode"); err != nil {
		return zero, err
	}

	var spec T
	if err := json.Unmarshal(trimmed, &spec); err != nil {
		return zero, fmt.Errorf("resources: decode %q spec: %w", c.kind, err)
	}
	if c.validator == nil {
		return spec, nil
	}
	validated, err := c.validator(ctx, scope, spec)
	if err != nil {
		return zero, fmt.Errorf("resources: validate %q spec: %w", c.kind, err)
	}
	return validated, nil
}

func validateCodecPayloadSize(size int, maxBytes int, kind ResourceKind, operation string) error {
	if size > maxBytes {
		return fmt.Errorf(
			"%w: %s %q payload exceeds %d bytes: %d",
			ErrPayloadTooLarge,
			operation,
			kind,
			maxBytes,
			size,
		)
	}
	return nil
}

type codecRegistration struct {
	specType reflect.Type
	codec    any
}

// CodecRegistry holds explicit kind-to-codec registrations for typed adapters.
type CodecRegistry struct {
	mu     sync.RWMutex
	codecs map[ResourceKind]codecRegistration
}

// NewCodecRegistry constructs an empty kind codec registry.
func NewCodecRegistry() *CodecRegistry {
	return &CodecRegistry{
		codecs: make(map[ResourceKind]codecRegistration),
	}
}

// ResolveCodec returns the typed codec registered for one resource kind.
func ResolveCodec[T any](registry *CodecRegistry, kind ResourceKind) (KindCodec[T], error) {
	if registry == nil {
		return nil, errors.New("resources: codec registry is required")
	}

	normalizedKind := kind.Normalize()
	if err := normalizedKind.Validate("kind"); err != nil {
		return nil, err
	}

	registry.mu.RLock()
	entry, ok := registry.codecs[normalizedKind]
	registry.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: codec not registered for kind %q", ErrCodecNotFound, normalizedKind)
	}

	codec, ok := entry.codec.(KindCodec[T])
	if !ok {
		return nil, fmt.Errorf(
			"%w: codec for kind %q is %s, not %s",
			ErrCodecTypeMismatch,
			normalizedKind,
			entry.specType,
			specTypeOf[T](),
		)
	}
	return codec, nil
}

// RegisterCodec adds one typed codec keyed by its resource kind.
func RegisterCodec[T any](registry *CodecRegistry, codec KindCodec[T]) error {
	if registry == nil {
		return errors.New("resources: codec registry is required")
	}

	normalizedKind, err := validateCodec(codec)
	if err != nil {
		return err
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.codecs[normalizedKind]; exists {
		return fmt.Errorf("%w: codec already registered for kind %q", ErrConflict, normalizedKind)
	}

	registry.codecs[normalizedKind] = codecRegistration{
		specType: specTypeOf[T](),
		codec:    codec,
	}
	return nil
}

func validateCodec[T any](codec KindCodec[T]) (ResourceKind, error) {
	if codec == nil {
		return "", errors.New("resources: codec is required")
	}

	normalizedKind := codec.Kind().Normalize()
	if err := normalizedKind.Validate("codec.kind"); err != nil {
		return "", err
	}
	if codec.MaxBytes() <= 0 {
		return "", fmt.Errorf("%w: codec.max_bytes must be positive: %d", ErrValidation, codec.MaxBytes())
	}
	return normalizedKind, nil
}

func specTypeOf[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}
