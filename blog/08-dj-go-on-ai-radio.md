# DJ Go on AI Radio

This is a casual internet radio written in Go. It streams music to many listeners, and an AI presenter makes short
introductions between the tracks.

These two jobs have very different needs. The music stream must keep a steady pace. The AI can be slow, fail, or say
nothing at all. The radio keeps the two paths apart and joins them with one hand-off that never waits, so the AI can never
stop the music.

This article follows how the system was built: how the streaming path is designed, which ideas were changed after
measuring them, and how the AI presenter is kept away from the music. The code is at
<https://github.com/volodymyr-pavlov/go-webrtc-audio>.

## What This Article Covers

The goal of the project was to see how far a plain Go program can go. On the path of the packets, "plain" means the
standard library, one WebRTC library (Pion), goroutines (Go's light threads), atomic operations and `io.Writer`. Around
that path, the server uses a few more libraries: fasthttp for the API, LangChain for Go for the model, and
`golang.org/x/net` for the batching socket. The server is one static program, with no kernel tuning and no kernel
bypass. The presenter adds four outside programs: `ffmpeg`, `ffprobe`, a speech program and an Ollama server.

The article is about design and code. It uses only a few numbers, and only where a number explains a decision. They come
from small tests on one cloud machine. They do not say anything about capacity.

The comparisons follow one rule. Each setting ran twice, in opposite orders, 20 seconds apart. If a difference was
smaller than the gap between the two runs of one setting, it was called noise. The steal time of the shared core was
read for every run. A first group of runs, made before it was read, was left out.

## Two Jobs, Two Speeds

Two facts shape the whole system.

1. A radio has one stream and many listeners. So the server should do almost all of its work once, not once for each
   listener.
2. The voice between the songs comes from a language model. A model is the least predictable part of the system, and it
   must never disturb the stream.

The diagram shows the two paths. The top path carries the music. The bottom path carries the DJ.

```text
  files ─▶ ffmpeg ─▶ packets (once) ─▶ playout ─▶ router ─▶ shards ─▶ listeners
                                          ▲
                                          │  one atomic pointer, a deadline
  facts ─▶ model ─▶ checked text ─▶ voice ┘
  (slow, unreliable, allowed to fail)
```

## The Playout: One Metronome

The aim is to give each listener as little new work as possible. So the design does every shared step before the stream
goes out to the listeners.

* The audio is encoded once, at start-up, into small packets of 20 milliseconds. The packets never change.
* One goroutine, the playout, sends the next packet every 20 ms, by the clock. If it is late, it catches up, so the
  stream does not drift. It catches up by five packets at most. That is 100 ms of audio with 20 ms packets, and 300 ms
  with 60 ms packets. After a longer stall, it skips the rest and counts the skip, because a listener cannot use audio
  older than its jitter buffer. The playout does not know who is listening. It only writes to an `io.Writer`.
* The writer is a router. It gives each packet to a few shards, one for each CPU core. A shard is a group of listeners.
  Its list of listeners is replaced as a whole and never edited, so reading it needs no lock.

What is left for each listener is the part that cannot be shared: its own encryption and its own send.

## No Allocations in the Project's Own Code

Giving each listener as little work as possible has a direct result in the code. The hot path is written to avoid
allocations. Allocations create garbage and more work for the GC, so the code avoids them where the packets flow.

* The Opus payloads are built once and shared by reference. Nothing copies them for each listener.
* The playout writes the RTP header and the payload into one buffer, and reuses it for every packet. A test checks that
  sending a packet makes no allocation.
* The router copies that framed packet once, so the playout's buffer cannot change the copy. The cost is the same for
  one listener and for fifty, and a test checks this. The work is per packet, not per listener.
* The batching socket reuses its datagram buffers and finds the address of a listener without allocating. A test checks
  the address lookup.
* The list of tracks is encoded once, and a test checks that reading it allocates nothing.
* In a small test on a Google Cloud `e2-micro` (two shared vCPUs, under 1 GB of memory), this code sent the stream to
  400 listeners over loopback. It used about a third of one core and 124 MB, and lost no packet. At 500 listeners, it
  lost about one packet in ten.

The claim has a limit. It covers the code of this project only. The encryption and the other work inside Pion were not
measured, so the whole path is not proven to be free of allocations.

## Go Habits That Keep It Simple

A few rules shaped the code in every package.

* **Small standard interfaces.** The playout writes to an `io.Writer`, and the router is that writer. Another consumer
  can be plugged in without any change to the playout.
* **Bit flags for states.** The state of an engine, a session or a playout is one integer with named bits. It is not a
  set of booleans and strings.
* **No lock on the packet path.** Counters, the playlist cursor and the lists of the shards are atomic values. The code
  declares no mutex. The tables away from the packets, such as the router's table of listeners, are a `sync.Map`.
* **Lists that are copied, not edited.** A new list replaces the old one in one step, and readers keep using the list
  they already hold.
* **Style kept by tools.** Three small analyzers, built on `go/analysis`, run as one `multichecker` binary. Every error
  is declared in `errors.go`. Each field of a composite literal has a line of its own. Each package-level `var` stands
  alone. The literals check can rewrite the code with `-fix`. `make fix-style` runs it six times, because one pass skips
  fixes that overlap.
* **Small packages.** Each package does one job and has a short doc comment: `playlist`, `stream`, `fanout`,
  `transport`, `engine`, `api`, `edge`, `metrics`, `config` and `ai`, among others.

## The Queue That Got Thrown Away

In the first design, each listener had its own small queue and its own writer goroutine. A slow listener lost only its
own packets, and it slowed nobody else. This is the usual answer to the slow listener problem, and it looked safe.

Then the sending became cheaper. On Linux, a batching socket takes the packets of all listeners into one shared ring (a
fixed queue). One goroutine sends a batch of them with one system call, `sendmmsg`. The flag `-batch` sets the size of a
batch, and the Docker image uses 64. Now a write to a listener is only a copy into the ring. The queue and the writer
for each listener only add one extra wake-up for each packet.

So the shard now writes to its listeners by itself. In a test with 100 listeners, this used 13 to 18 percent less CPU.
That is more than the gap between the two runs of one setting.

Without a queue for each listener, something else must remove a listener that fails. The router counts, for each
listener, the writes in a row that went wrong. When the count reaches a limit, the router evicts the listener once and
calls the server's callback on a new goroutine. One good write resets the count. Nothing in this path ever waits.

The change has a cost, and the cost depends on the setting.

* With batching on, UDP never waits for a listener, so a slow listener cannot slow the sender. The ring can still fill
  up if the sender falls behind the socket. Then the packets of whoever pushes next are dropped, not the packets of the
  one that caused it.
* With batching off and no queue, a shard writes to Pion directly. A send that blocks would hold up every listener of
  that shard.

Neither case was tested. That is why the server still keeps the queues by default when batching is off.

A safety measure should be removed only after its cost is measured, and the notes should say what it protected.

## Inside the Batching Socket

Three details of the batching socket are worth a note.

The ring is a lock-free queue, designed by Dmitry Vyukov. Every slot has a sequence number that says whose turn it is.
A push or a pop claims its place with a compare-and-swap. The head and the tail sit on separate cache lines, because the
writers touch one and the sender touches the other.

The sender goroutine sleeps when the ring is empty, and a lost wake-up can hide there. A writer may push just after the
sender saw an empty ring, and just before it went to sleep. Then nobody wakes the sender. So the sender first sets a
"sleeping" flag, and then looks at the ring once more. A writer that pushes clears the flag and wakes the sender.
Whoever clears the flag does the waking, so there is one wake-up and never none.

The sending has one trap. `WriteBatch` returns -1 when it sent nothing, for example when the socket has no route to the
address of one listener. The loop moves on by the number of packets sent, so with -1 it would send the same batch for
ever. The fix is to count the failed packet as sent. The packet at the head is dropped, because UDP does not retry, and
the loop goes on with the rest.

Even so, the batching itself gave only 4 to 9 percent at 100 listeners. That is less than the gap between the two runs
of one setting, so it counts as noise. What the batching socket bought was the right to remove the queues.

## Dead Ends and the 60 ms Trade-off

Two other ideas were tried, and both gave less than expected. A separate playout for each listener used more CPU than
the router. One shared Pion track used less CPU, but then one blocked write can hold up every listener, so the router
stayed.

The idea that won is a product choice, not a trick. A packet can hold 20, 40 or 60 ms of audio. With 60 ms packets, the
CPU use fell by 57 percent. The price is up to 40 ms more delay, and a longer gap when a packet is lost. The default is
still 20 ms, the usual Opus size that every client handles. The flag `-frame` changes it.

## The Llama Gets a Deadline

The second part of the system is the presenter. It lives inside the server, in its own package. It has one job: while a
track plays, it says something about the next track. It works in five steps, and the model is only step three.

1. **Facts.** The tool `ffprobe` and a loudness check give the next track its length, loudness and credits.
2. **Text.** A prompt template, built with LangChain for Go, asks a local model for a sentence or two. The model is
   `phi3.5`, running in Ollama (the llama on the cover stands for it). The prompt has worked examples, because a small
   model often ignores a plain rule but follows an example.
3. **Check.** The answer is treated as untrusted text. It must be one plain line, with no links or markup. Numbers must
   come only from the facts, and no artist may be invented. If the check fails twice, the server builds a plain line
   without the model.
4. **Voice.** A speech program reads the line, and `ffmpeg` encodes the sound.
5. **Splice.** The finished clip goes into an atomic slot. This is a pointer that changes in one step, without a lock.
   At the start of a track, the playout looks at the slot and plays the clip before the track.

<figure class="feature-figure">
<img class="feature" src="../images/08-radio-app.jpeg" alt="The radio page while the presenter speaks: a player card with a LIVE badge, a Stop button, the track on the air, a violet AI PRESENTER panel with a SPEAKING mark and the words 'Up next, third track, calm in essence.', the connection figures, and a list of three tracks." />
<figcaption>The app while the presenter speaks: the track on the air, the AI presenter panel with its SPEAKING mark and the words it is saying, and the track list. The tracks here are standard test tones.</figcaption>
</figure>

The most important rule is in the last step. The presenter runs on its own goroutines. The only thing it gives to the
playout is one atomic pointer. Every introduction has a deadline: the length of the current track minus three seconds.
If a clip is late, the server counts it. The track starts without it, and the stream does not stop. The listener's page
does not show the miss.

One case is rarer. After a stall, the playout may skip audio, and a track can then end before its deadline. A clip that
is finished after that waits until the playlist comes round to its track again.

This pattern works in other projects too. Sometimes a part that must never stop meets a part that can fail. Join them in
one place with one hand-off that never waits. Give the unreliable part a deadline, and let it fail quietly. A slow
model, a broken speech program or a busy machine all end the same way: a track with no introduction.

## How It Is Tested

The tests follow the same split between the hard part and the soft part.

* **The playout** has a test that checks it sends one packet every 20 ms. The clock rule is checked, not assumed.
* **The presenter** is tested without a real model. A fake model server and a fake speech program return good text, bad
  text and nothing at all. The tests check the deadline, a track that is too short, the fallback line, a failing model
  or voice, and a newer clip that replaces an older one.
* **Allocations.** `testing.AllocsPerRun` checks the playout's send step, the router's copy, the address lookup of the
  batching socket and the list of tracks.
* **Races and leaks.** The test command runs the Go race detector. Several packages also check that no goroutine is left
  behind after a test.
* **The browser.** Automated Chromium tests run against a real backend: listening, reconnecting after the backend
  restarts, a closed tab, Stop, and a connection that may only use a relay.

## Three Things to Remember

1. **Do the work once.** Anything that can be done for the stream, and not for each listener, belongs before the
   sending.
2. **Measure before removing a safeguard.** The queue for each listener looked necessary, and it was not. This was known
   only after the cheaper send was measured against it.
3. **One hand-off, one deadline.** That is all the two parts of the radio share. It is why a slow, wrong or missing
   presenter leaves the music playing.

