# Events, Outbox and Jobs

- Module ID: `MOD-EVT`
- Roadmap topic: `V020-T05`
- Implementation state: architecture scaffold only

Owns outbox entries, delivery attempts, background jobs, notification requests, and delivery status. It consumes versioned events and propagates committed facts.

Delivery, retry, quarantine, or replay cannot directly change another module's authoritative business state.
