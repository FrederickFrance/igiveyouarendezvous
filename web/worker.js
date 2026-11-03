// The planner runs in this worker, so that the page stays responsive while
// it searches: the Go program, compiled to WebAssembly, sets the global
// function igiveyouarendezvous (cmd/igiveyouarendezvous/main_js.go).
importScripts("wasm_exec.js");

const go = new Go();

const ready = (async () => {
  let result;
  try {
    result = await WebAssembly.instantiateStreaming(fetch("igiveyouarendezvous.wasm"), go.importObject);
  } catch {
    // A server that does not send the type application/wasm.
    const response = await fetch("igiveyouarendezvous.wasm");
    if (!response.ok) {
      throw new Error(`igiveyouarendezvous.wasm: ${response.status} ${response.statusText}`);
    }
    result = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
  }
  go.run(result.instance); // returns once main waits for calls
  postMessage({ type: "ready" });
})().catch((error) => postMessage({ type: "failed", message: String(error) }));

onmessage = async (event) => {
  await ready;

  const { problem, args, inputs } = event.data;
  const result = await self.igiveyouarendezvous(
    problem,
    args,
    inputs,
    (line) => postMessage({ type: "line", line }),
    (text, plan) => postMessage({ type: "best", text, plan }),
  );

  postMessage({ type: "done", result });
};
