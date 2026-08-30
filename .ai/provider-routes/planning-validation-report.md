# P0-C AI Service Route Control Validation Report

**Validation date:** 2026-08-18

## Result

All generated planning-control artifacts passed local schema and cross-reference validation. No provider route or model is enabled.

## Validated artifacts

- `ai-service-route-registry.yaml` against `schemas/ai-service-route-registry.schema.json` — PASS
- `model-registry.yaml` against `schemas/model-registry.schema.json` — PASS
- `provider-routing.yaml` against `schemas/provider-routing.schema.json` — PASS
- `budgets.yaml` against `schemas/ai-service-usage-controls.schema.json` — PASS
- `ai-service-usage-register.yaml` against `schemas/ai-service-usage-register.schema.json` — PASS
- `Provider Data Policy Review Example.yaml` against `schemas/provider-data-policy-review.schema.json` — PASS

## Cross-checks

- Provider route IDs are unique: 6.
- Model record IDs are unique: 6.
- Every route references an existing model record.
- `enabled_route_ids` is empty.
- `enabled_model_record_ids` is empty.
- Every candidate route has `dispatch_enabled: false`.
- Every model record has `dispatch_enabled: false`.
- Every canonical role route is unassigned and disabled.
- Fixed budget is not established; currency and monetary limits remain unknown.
- Legacy brand-level routing aliases are marked unqualified and non-dispatchable.

## Limitations

- Validation proves document and schema consistency only.
- No provider terms, account, model, CLI, adapter, quota, cost, technical behavior, security control, or evaluation result has been qualified.
- Runtime enforcement in Herdr/herdctl is not implemented.
- No GitHub action was performed.

## Artifact digests

- `00 AI Service Route Registry Package Index.md` — SHA-256 `0ec00c1bbf5e9d5d2e11281991702f2308dce4d3005642c002cf152fdf100049`
- `00 AI Service and Provider Route Control Package Index.md` — SHA-256 `190f8308c3c9c93b10929acc94bd5c8d3e7352a73f9f2d759dfe05b8c8d5a987`
- `AI Service Candidate Routing Quota and Failover Matrix.md` — SHA-256 `2f6cfda95454ea02f986df378bd8c9c481feef6a4849250cba9db8f399d6f264`
- `Provider Data Policy Review Example.yaml` — SHA-256 `0393812021c38f1e36448a742fa612e3e2b11aaa50603260134451841093feb4`
- `Provider Data Policy Review Template.md` — SHA-256 `7063c38448ea6fb76c576557e9c9fbd55f18f4e4f31e7bc6d7bfdd44d6295b7d`
- `Provider Route Intake Example.yaml` — SHA-256 `fc972a4e75e910b3fcb946bd80b5b97b62ca9fd33abfaf2b3c38726832c8edd1`
- `ai-service-route-registry.yaml` — SHA-256 `63b45e53fe03f2b5ad98b1aa20f6de04552c37a44fff13486758ca08088b1df2`
- `ai-service-route.schema.json` — SHA-256 `df30e2e852625cfd6748ff8e2bc9f93c652371926a18e2e5b259d0f93eaae798`
- `ai-service-usage-controls.schema.json` — SHA-256 `fc15895c1cc8d0791d8b009f879998b35f11c85f6e2198628a64d734e7c58075`
- `ai-service-usage-register.schema.json` — SHA-256 `27a2e6b4a4443bd8475f2c6530115224f4395624cb72fb46f0181f9d8b1c9968`
- `ai-service-usage-register.yaml` — SHA-256 `9a70dcbf29245301b11c63f22cae3f4b3bd79e1a3b349d1cea2eb451c0ebcfe6`
- `budgets.yaml` — SHA-256 `2b6e1ae7cb57b1d5e02e8a6b4f8ac91afd317c0e8d18601eb410ec8ca39731b2`
- `data-classification.yaml` — SHA-256 `576ab1494e4172cb5d7fea3cf67737d61cc4a5a08d2cc1668e2a79d7f281b32d`
- `model-registry.schema.json` — SHA-256 `5506395130449f775bbf8f0a6ba203161a3ce9fc2e15d74238fc24d8aba0c9d2`
- `model-registry.yaml` — SHA-256 `ab3de20fa985534d69ff8a4129a24f7b7775df9a0b05b59ff98d5acebdd325a7`
- `provider-data-policy-review.schema.json` — SHA-256 `055fdc2dbe1f31ae99f5d50ca320db92e41ae6a1f1bbe9c839a78b762b19cbba`
- `provider-policy-review-register.yaml` — SHA-256 `90ac0617aef7cab37a03c9b07f98948e2842aa88e58ed69263746ae5cd3a5f79`
- `provider-routing.schema.json` — SHA-256 `d04579fac6489403145c6694425f2f7d901ccc7dfbad35a33902723307a31345`
- `provider-routing.yaml` — SHA-256 `82559dafb8529d77619c70970154fab4c98a080af35240b2b9f831c35bc20004`
