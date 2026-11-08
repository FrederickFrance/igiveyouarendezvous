package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// runInMemory runs the command on a problem given as text, as the online
// demo does in the browser, which has no files: the files the command reads
// (a -start plan) come from inputs, by name, and those it writes (-out,
// -rest) are returned, by name, with what it prints. best, if not nil,
// receives every better schedule found, as printed, as a JSON plan and, with
// -csv, as a table.
func runInMemory(ctx context.Context, args []string, problem string, inputs map[string]string, stderr io.Writer,
	best func(text, plan, table string),
) (string, map[string]string, error) {
	outputs := memory{}

	var out strings.Builder

	err := runOn(ctx, files{read: memory(inputs).read, write: outputs.write, best: best}, args, strings.NewReader(problem), &out, stderr)

	return out.String(), outputs, err
}

// memory is a set of files kept in memory, by name.
type memory map[string]string

func (m memory) read(path string) ([]byte, error) {
	text, ok := m[path]
	if !ok {
		return nil, fmt.Errorf("open %s: %w", path, fs.ErrNotExist)
	}

	return []byte(text), nil
}

func (m memory) write(path string, write func(io.Writer) error) error {
	var b strings.Builder
	if err := write(&b); err != nil {
		return err
	}

	m[path] = b.String()

	return nil
}

// lineWriter calls line with each line written to it, without its end;
// Flush passes on the last one when it has no end.
type lineWriter struct {
	line func(string)
	rest []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.rest = append(w.rest, p...)

	for {
		i := bytes.IndexByte(w.rest, '\n')
		if i < 0 {
			return len(p), nil
		}

		w.line(string(w.rest[:i]))
		w.rest = w.rest[i+1:]
	}
}

// Flush passes on the last line, when it has no end.
func (w *lineWriter) Flush() {
	if len(w.rest) > 0 {
		w.line(string(w.rest))
		w.rest = nil
	}
}
