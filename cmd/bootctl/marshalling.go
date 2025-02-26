package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func primitiveUnmarshaller[T any](r io.Reader) (out T, err error) {
	err = binary.Read(r, binary.LittleEndian, &out)
	return
}

func primitiveMarshaller[T any](w io.Writer, inp T) error {
	return binary.Write(w, binary.LittleEndian, inp)
}

func sliceUnmarshaller[T any](r io.Reader) (out []T, err error) {
	for i := 0; ; i += 1 {
		var item T
		err = binary.Read(r, binary.LittleEndian, &item)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			} else {
				err = fmt.Errorf("item #%d: %w", i, err)
			}
			return
		}
		out = append(out, item)
	}
}

func sliceMarshaller[T any](w io.Writer, inp []T) (err error) {
	var buf bytes.Buffer

	for i, item := range inp {
		err := binary.Write(&buf, binary.LittleEndian, item)
		if err != nil {
			return fmt.Errorf("item #%d: %w", i, err)
		}
	}

	_, err = buf.WriteTo(w)
	return
}

type readerFrom[T any] interface {
	io.ReaderFrom
	*T
}

func structUnmarshaller[T any, PT readerFrom[T]](r io.Reader) (out *T, err error) {
	var value T
	_, err = PT(&value).ReadFrom(r)
	if err == nil {
		out = &value
	}
	return
}

func structMarshaller[T io.WriterTo](w io.Writer, inp T) (err error) {
	_, err = inp.WriteTo(w)
	return
}
