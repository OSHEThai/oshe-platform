---
name: observability-telemetry-change
description: >
  Design logs, metrics, traces, health signals, alert context, privacy filters, and evidence attribution. Use when behavior must be observable or diagnosable.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# Observability and Telemetry Change

## Objective

Make behavior diagnosable without leaking sensitive data or confusing synthetic signals with production evidence.

## Required Inputs

- user journey, service boundaries, failure modes, data classes, and SLO intent;
- logging, metric, trace, health, retention, and access constraints;
- test environment and evidence class.

## Procedure

1. Define the questions each signal must answer and its owner.
2. Specify stable event names, dimensions, correlation, sampling, and timestamps.
3. Remove secrets, personal data, and high-cardinality unsafe fields.
4. Add health and failure signals with actionable context and bounded retention.
5. Validate signal emission and label the evidence environment accurately.

## Required Output

Signal contract, privacy review, tests, sample evidence, limitations, and operational decisions needed.

## Stop Conditions

- a proposed signal exposes prohibited data;
- retention or access authority is undefined;
- synthetic output is being presented as runtime proof.

## Evaluation Cases

- accept correlated redacted signals with bounded dimensions;
- reject secret-bearing logs or evidence without environment identity.
