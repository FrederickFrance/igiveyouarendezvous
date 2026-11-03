//go:build js && wasm

package main

import (
	"context"
	"errors"
	"flag"
	"syscall/js"
)

// main makes the command callable from JavaScript, for the online demo
// (web/): it sets the global function
//
//	igiveyouarendezvous(problem, args, inputs, progress, best) → Promise<result>
//
// problem is the text of the problem file, args the flags of the command,
// inputs an object of the files it may read, by name (a -start plan),
// progress a function called with each line printed on the standard error
// output (notes, progress), and best a function called with every better
// schedule found, as printed and as a JSON plan. The result has the printed
// schedule (output), the files written (files, by name), and error or alerts
// when there are some.
func main() {
	js.Global().Set("igiveyouarendezvous", js.FuncOf(plan))

	select {} // serve the calls until the page closes
}

func plan(_ js.Value, a []js.Value) any {
	problem, args, inputs, progress, best := a[0].String(), goStrings(a[1]), map[string]string{}, a[3], a[4]

	keys := js.Global().Get("Object").Call("keys", a[2])
	for i := range keys.Length() {
		name := keys.Index(i).String()
		inputs[name] = a[2].Get(name).String()
	}

	// A function called from JavaScript must not block: the search runs in
	// a goroutine, and the promise is resolved at its end.
	var executor js.Func

	executor = js.FuncOf(func(_ js.Value, p []js.Value) any {
		resolve := p[0]

		go func() {
			defer executor.Release()

			stderr := &lineWriter{line: func(line string) { progress.Invoke(line) }}
			output, written, err := runInMemory(context.Background(), args, problem, inputs, stderr,
				func(text, plan string) { best.Invoke(text, plan) })
			stderr.Flush()

			produced := map[string]any{}
			for name, text := range written {
				produced[name] = text
			}

			result := map[string]any{"output": output, "files": produced}

			switch {
			case errors.Is(err, errAlerts): // reported with the schedule
				result["alerts"] = true
			case err != nil && !errors.Is(err, flag.ErrHelp):
				result["error"] = err.Error()
			}

			resolve.Invoke(js.ValueOf(result))
		}()

		return nil
	})

	return js.Global().Get("Promise").New(executor)
}

// goStrings converts a JavaScript array of strings.
func goStrings(v js.Value) []string {
	out := make([]string, v.Length())
	for i := range out {
		out[i] = v.Index(i).String()
	}

	return out
}
