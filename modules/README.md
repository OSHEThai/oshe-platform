# Business Modules

Each module owns its state, schema, rules, application services, internal implementation, and versioned public contracts. Module creation requires a manifest and architecture tests.

The directories in this foundation are architecture scaffolds. They do not contain application source code and do not activate a module. Implementations, manifests, migrations, and tests are added through their roadmap topics after the toolchain is frozen.

Cross-module writes and direct access to another module's private tables or internal types are prohibited. Integrations use reviewed, versioned public commands, queries, assertions, references, snapshots, or events.

The authoritative scaffold list is maintained in `module-registry.yaml`.
