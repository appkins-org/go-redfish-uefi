package main

import (
	"bytes"
	"fmt"
	"io"

	"github.com/0x5a17ed/uefi/efi/efiguid"
)

const (
	globalAccess = BootServiceAccess | RuntimeAccess

	defaultAttrs = NonVolatile | globalAccess
)

type MarshalFn[T any] func(w io.Writer, inp T) error
type UnmarshalFn[T any] func(r io.Reader) (T, error)

type Variable[T any] struct {
	name         string
	guid         efiguid.GUID
	defaultAttrs Attributes

	marshal   MarshalFn[T]
	unmarshal UnmarshalFn[T]
}

func (e Variable[T]) Get(c Context) (attrs Attributes, value T, err error) {
	if e.unmarshal == nil {
		err = fmt.Errorf("efivars/get(%s): unsupported", e.name)
		return
	}

	attrs, data, err := ReadAll(c, e.name, e.guid)
	if err != nil {
		err = fmt.Errorf("efivars/get(%s): load: %w", e.name, err)
		return
	}

	value, err = e.unmarshal(bytes.NewReader(data))
	if err != nil {
		err = fmt.Errorf("efivars/get(%s): parse: %w", e.name, err)
	}
	return
}

func (e Variable[T]) SetWithAttributes(c Context, attrs Attributes, value T) error {
	if e.marshal == nil {
		return fmt.Errorf("efivars/set(%s): unsupported", e.name)
	}

	var buf bytes.Buffer
	if err := e.marshal(&buf, value); err != nil {
		return fmt.Errorf("efivars/set(%s): write: %w", e.name, err)
	}
	return c.Set(e.name, e.guid, attrs, buf.Bytes())
}

func (e Variable[T]) Set(c Context, value T) error {
	return e.SetWithAttributes(c, e.defaultAttrs, value)
}
