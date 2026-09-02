---
provider_id: "qwen"
provider_name: "Qwen"
note_mode: "STATIC_FAIL_CLOSED"
supported_context_entry_points:
  - "AGENTS.md"
  - "QWEN.md"
  - ".ai/provider-notes/qwen.md"
output_result_boundary: "ASSIGNED_OUTPUT_CONTRACT_ONLY_NO_AUTHORITY"
secret_handling: "PROHIBITED"
customer_data_handling: "PROHIBITED"
route_status: "DEFAULT_DENY_NO_APPROVED_ROUTE"
unsupported_invocation: "FAIL_CLOSED_NO_DISPATCH"
active_behavior_owners:
  adapter_runtime: "V010-I022"
  provider_model_data_policy_route: "V010-I023"
  quota_budget_failover: "V010-I024"
approved_credential: "NONE"
model_alias_selection: "NONE"
retention_promise: "NONE"
numeric_budget: "DEFERRED_BY_HDEC_037"
smoke_test_claim: "NONE"
reserved_root_file: "NONE"
reserved_root_file_status: "NOT_APPLICABLE"
---
# Qwen Provider Notes

## Identity and supported context

The provider identity is `qwen` and its display name is Qwen. The supported static context entry points are exactly those listed in the frontmatter. They convey context only and grant no execution authority.

## Output and data boundary

Output is limited to the assigned output contract and is never human approval, route authority, acceptance, merge authority, or release authority. Secrets and customer data are prohibited.

## Default-deny behavior

The route remains default-deny with no approved route. An unsupported invocation must fail closed without dispatch; a provider name alone never resolves a route or assignment. A local executable does not bypass any control.

## Later active-behavior owners

V010-I022 owns adapter launch, version detection, output markers, and adapter failure behavior. V010-I023 owns provider/model registry entries, allowed data classes, retention review, and route enablement evidence. V010-I024 owns quota, budget, concurrency, reserve, and failover enforcement.

## Prohibited static claims

This note does not enable dispatch, name an approved credential, select a model alias, promise retention behavior, set a numeric budget, or claim a smoke test. The numeric adapter/context budget remains `DEFERRED_BY_HDEC_037`.

## Reserved root-file boundary

HDEC-037 assigns no reserved root file to this static provider note.
