# Circuit Breaker Go

A Go implementation of the circuit breaker pattern with state tracking, half-open recovery, context-aware calls, and a live React dashboard.

---

## What problem does it solve?

Imagine you're using an app to buy something online. Behind the scenes, that app talks to dozens of other services — one to check your payment, one to check stock, one to send a confirmation email, and so on.

Now imagine one of those services — say, the payment service — starts having problems and takes 30 seconds to respond instead of the usual half a second. Without any protection, every user clicking "Buy" would be stuck waiting, the app would pile up thousands of frozen requests, and eventually the whole thing could crash.

A **circuit breaker** is the fix for that. It works exactly like the circuit breaker in your home's fuse box. When something goes wrong and too much current flows through, the fuse trips and cuts the power — protecting everything else. Once the problem is fixed, you reset it and power comes back.

This project is a circuit breaker built in Go, with a live dashboard so you can watch it work in real time.

---

## How does it work?

The breaker has three states:

### 🟢 Closed — everything is normal
Requests flow through freely. The breaker is quietly counting how many are failing. If failures stay below the threshold, nothing changes.

### 🔴 Open — something is wrong, stop trying
Once too many failures happen in a row (3 by default), the breaker trips open. Instead of sending more requests to the broken service and making things worse, it immediately returns an error. This gives the broken service time to recover without being bombarded.

### 🟡 Half-Open — let's test if it's fixed
After a timeout (5 seconds by default), the breaker cautiously lets a small number of requests through. If they succeed, it assumes the service is healthy again and goes back to Closed. If they fail, it trips back to Open and waits again.

### A real-world analogy

Think of it like a bouncer at a club.

- **Closed**: The door is open, everyone gets in normally.
- **Open**: There's a fight inside. The bouncer closes the door and turns everyone away immediately rather than letting more people in to make it worse.
- **Half-Open**: After a while, the bouncer opens the door a crack and lets one or two people in to check if things have calmed down. If it's fine, the door opens fully again. If not, it shuts again.

---

## What's in the project?

| Part | What it does |
|---|---|
| `circuit/` | The core logic — state machine, failure counting, recovery |
| `main.go` | A small web server that exposes the breaker over HTTP |
| `ui/index.html` | A live dashboard built with React to visualise everything |

---

## Features

- Closed / Open / Half-Open states
- Custom `ReadyToTrip` logic
- `Timeout` for open-state recovery
- `Interval` support for expiring stale consecutive failures
- Context cancellation support during calls
- `OnStateChange` callback with safe lock handling
- Unit tests covering open/recovery, half-open cancellation, interval reset, callback order, and concurrency

---

## Run

```bash
go run .
```

Then open `http://localhost:8080` in your browser to view the React dashboard.

## Test

```bash
go test ./circuit
```

---

## The dashboard

When you open the app in your browser, you see:

- **A live state badge** — green, yellow, or red depending on the breaker's current state, with a pulsing dot
- **Metrics** — how many requests have been made, how many succeeded, how many failed, and how many failures happened in a row
- **Controls** — buttons to simulate a successful call or a failing call, so you can watch the breaker trip and recover without needing a real broken service
- **Burst mode** — fire up to 20 calls at once with a slider, so you can trip the breaker quickly instead of clicking one at a time
- **State history** — a row of coloured dots showing every state the breaker has been in since the page loaded
- **Event log** — a timestamped feed of everything that has happened, colour-coded by outcome

---

## What it is and isn't

**It is** a clean, working implementation of the circuit breaker pattern — the same pattern used by large companies like Netflix, Amazon, and Google to keep their systems stable when parts of them fail.

**It isn't** a production-ready drop-in tool yet. Right now it simulates fake calls rather than wrapping real services, and its state lives only in memory (restart the server and it resets). To use it in a real system you'd wire it around actual database or API calls, and hook its metrics into your monitoring setup.

---

## Why does this pattern matter?

In modern software, nothing runs alone. Every app depends on other services, and those services depend on others. Failures cascade — one slow service can take down an entire platform if there's nothing to stop the chain reaction.

The circuit breaker pattern is one of the most important tools for building systems that **fail gracefully** — meaning when something goes wrong, the damage stays contained and the rest of the system keeps working.
