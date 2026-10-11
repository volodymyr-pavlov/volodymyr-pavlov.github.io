# Playing Go Using Go

Here is a small game of Go you can play in your browser, and the opponent is a Monte Carlo Tree Search engine written
in... Go. No server thinks for it. The engine is compiled to WebAssembly, lives in a Web Worker,
and plays up to a thousand random games before it answers. This article is about the Wasm part: how to build it, how
to call it from JavaScript, and what it took to keep the whole thing small and fast. You can
[play it here](https://go-game.volodymyr-pavlov.workers.dev/) first, if you like.

## Why Wasm at All?

The rules of Go are tiny, but a playable opponent needs to simulate thousands of games per move. That is a hot loop:
flood fills, liberty counts, captures, over and over. The goal was to write that loop in Go, with tests, benchmarks
and a profiler, and not in JavaScript. WebAssembly makes that possible: compile the Go package once and ship it as a
binary the browser runs at close to native speed.

The rest of the game is plain TypeScript and a canvas. The engine is one Go package and one `main.go`.

## The Build Is One Line

Go has shipped a Wasm target since 1.11. You do not need a plugin or a toolchain, just two environment variables:

```sh
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o public/engine.wasm .
```

That gives you a `.wasm` file. To run it you also need `wasm_exec.js`, a small shim that ships with Go and teaches the
browser how to start a Go program. Since Go 1.24 it lives in `lib/wasm`:

```sh
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" public/
```

One trap: the shim and the binary must come from the same Go version. The shim is the runtime's other half, and a
mismatch fails at load time with an import error that says nothing about versions. Copy it in the same script that
builds the binary, every time.

The `-s -w` flags strip symbols and DWARF. Without them the binary is noticeably bigger, and nobody debugs a Wasm
binary with `gdb` anyway. This one lands at about 4.7 MB, which is mostly the Go runtime and the garbage collector.
More on that below.

## The Go Side: One Exported Function

The Wasm entry point is a normal `main` package with a build tag so it never gets compiled by accident on your laptop:

```go
//go:build js && wasm

package main

import (
    "math/rand/v2"
    "syscall/js"

    "github.com/volodymyr-pavlov/go-game/packages/engine/game"
)

func main() {
    js.Global().Set("suggestMove", js.FuncOf(suggestMove))
    select {} // keep the module alive for further calls
}
```

`js.Global().Set(...)` puts a function on the JavaScript global object. That is the whole API: one function. Everything
else stays private to Go.

`select {}` blocks forever. If `main` returns, the Go runtime exits and every exported function is gone. Blocking on an
empty select keeps the program alive, parked, using no CPU, until JavaScript calls in.

The function itself takes a JSON string and a number, and returns a JSON string:

```go
func suggestMove(_ js.Value, args []js.Value) any {
    if len(args) != 2 {
        return game.ToJSON(game.ErrorResponse{Error: "suggestMove expects (positionJSON, rollouts)"})
    }
    position, err := game.ParsePosition(args[0].String())
    if err != nil {
        return game.ToJSON(game.ErrorResponse{Error: err.Error()})
    }
    if args[1].Type() != js.TypeNumber {
        return game.ToJSON(game.ErrorResponse{Error: "rollouts must be a number"})
    }
    rollouts := game.ClampRollouts(args[1].Int())
    result := game.Search(position, rollouts, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
    return game.ToJSON(game.ToResponse(result))
}
```

### Why JSON strings and not objects?

`syscall/js` can read JavaScript objects field by field, but every `Get` is a call across the Wasm boundary, and each
one costs more than reading a byte. A 9x9 board is 81 cells, plus ko, captures and who is to play. Passing that as
one string and parsing it in Go is one boundary crossing instead of a hundred. Returning a string works the same way.

It also keeps the contract simple. The TypeScript side sends the same `Position` object it uses everywhere, serialised
with `JSON.stringify`, and gets back `{ move, winRate, visits }` or `{ error }`. There is no shared memory and no
custom ABI to maintain, and when the engine fails it fails with a readable message.

The parser does not trust the request. The Wasm module is public, and anyone can call `suggestMove` from the browser
console. So it checks that the board is square, that every cell is 0, 1 or 2, that the ko point is on the
board, and it clamps the number of rollouts so a hand-made request cannot freeze the tab.

## The JavaScript Side: Load, Run, Call

Loading a Go Wasm module is three steps: import the shim, instantiate the binary with the shim's imports, and start
the Go program.

```ts
await import('/wasm_exec.js');
const go = new Go();
const response = await fetch('/engine.wasm');
const { instance } = await WebAssembly.instantiateStreaming(response, go.importObject);
go.run(instance);
```

`go.run` returns a promise that resolves when the Go program exits. Thanks to `select {}` that never happens, unless
the program panics or exits. The loader keeps that promise around, and if it ever settles, every later call throws a
clear error instead of calling a function that no longer exists:

```ts
let exit: Error | null = null;
go.run(instance).then(
  () => (exit = new Error('The Go engine has stopped')),
  (error) => (exit = new Error(String(error))),
);

return {
  suggestMove: (positionJson: string, rollouts: number): string => {
    if (exit !== null) throw exit;
    return suggestMove(positionJson, rollouts);
  },
};
```

`instantiateStreaming` compiles the binary while it is still downloading, but it only works when the server sends
`Content-Type: application/wasm`. Cloudflare does. A local dev server might not, so the loader falls back to
`arrayBuffer()` plus `WebAssembly.instantiate` when the header is missing.

## Put It in a Worker

A thousand playouts take a few hundred milliseconds. If that runs on the main thread, the board freezes, the stone
you just placed does not animate, and the page feels broken. So the engine lives in a Web Worker:

```ts
const worker = new Worker(new URL('./engine.worker.ts', import.meta.url), { type: 'module' });
worker.postMessage({ type: 'init', wasmUrl: '/engine.wasm' });
worker.postMessage({ type: 'search', id: 1, position, rollouts: 1000 });
worker.onmessage = ({ data }) => { /* { type: 'suggestion', id, suggestion } */ };
```

The worker loads the Wasm module once and answers requests by id. The main thread never waits. It sends a position,
keeps drawing, and places the engine's stone when the answer arrives. The Go side is single-threaded and does not know
it is in a worker.

## Making the Hot Loop Fast

Wasm is fast, but the Go garbage collector still runs inside it, and in a browser every GC pause is time the user
waits. The fix is the same as on a server: allocate nothing in the hot path.

* **A neighbour table by index.** Every point on the board gets an index, and a table built once per board size lists
  its neighbours. No `Point` structs and no map lookups in the loop, just `int32` slices.
* **A reusable flood-fill scratch.** Counting liberties means flooding a group. Instead of a fresh `map` or slice per
  call, one `Scratch` struct holds a mark array and a stack. A generation counter says which marks are current, so
  clearing the array is one increment, not a loop.
* **Playouts in place.** Each random game is played on one board that is reset between playouts, not copied per move.

None of this is Wasm specific. The loop was written, benchmarked and profiled with the normal Go toolchain on a laptop,
and the browser got the same speedup. `go test -bench` works, `pprof` works, and then you build with `GOOS=js`.

## What About the 4.7 MB?

The Go runtime is not small, and all of it ends up in the binary. A few things help:

* **Strip it.** `-ldflags="-s -w"` is the cheapest win.
* **Compress it.** Wasm compresses well, and who compresses it matters. Cloudflare compresses on the way out, once
  per response, so it picks a fast, low Brotli level: the 4.7 MB engine went over the wire as 1.3 MB with gzip and
  1.2 MB with Brotli. Firebase Hosting, where this game lived before, compressed once at deploy time at the highest
  level and served the same file as 0.96 MB. A host that compresses at request time trades bytes for latency. So the
  build now compresses the binary itself, once, at Brotli level 11, and a twenty-line Worker in front of the static
  site hands that `.br` file to any browser that accepts Brotli, with `encodeBody: "manual"` so Cloudflare does not
  compress it again. The engine is 0.96 MB on the wire again, and the Worker runs only for that one path.
* **Load it late.** The worker is not started until the engine's first turn. The board is drawn and ready to play
  before a single byte of the engine is downloaded.
* **Cache it.** The binary is served as plain `engine.wasm` with Cloudflare's default `Cache-Control: public, max-age=0,
  must-revalidate`, so the browser asks the server on every visit and downloads the file again only when it changed.
  That is safe for updates but not free. For a long-lived cache, put a hash or a version in the file name and set a
  long `max-age`.

If size really matters, TinyGo produces binaries an order of magnitude smaller, but it comes with a different garbage
collector, a subset of the standard library and a slower compile. For a game that people open once and play for ten
minutes, the standard toolchain was the right trade.

## The Whole Picture

A move travels like this:

1. You place a stone. TypeScript updates the position and sends it to the worker.
2. The worker calls `suggestMove(JSON.stringify(position), rollouts)` on the Go module. Gopher asks for 200 rollouts,
   Bear for 500 and Elephant for 1000.
3. Go parses the JSON, runs the playouts without allocating, picks the most visited move, and returns JSON.
4. The worker posts the answer back, and the board animates the engine's stone.

One Go package, one exported function, one worker, a string in and a string out. That was enough to ship a game
engine to the browser without a server, and to keep writing it in Go.

If you beat Elephant, there is confetti.
