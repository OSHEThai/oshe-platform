---
name: accessibility-usability-review
description: >
  Review accessible interactions, content, keyboard and assistive-technology behavior, and human usability evidence gaps. Use for user-facing flows and content.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# Accessibility and Usability Review

## Objective

Identify accessibility and usability barriers while distinguishing automated checks from real-user evidence.

## Required Inputs

- target journeys, supported devices, languages, design and content sources;
- applicable accessibility standard and acceptance criteria;
- rendered or running artifact and test environment.

## Procedure

1. Map critical journeys, states, errors, timeouts, and safety-related messages.
2. Review semantics, keyboard path, focus, contrast, scaling, labels, errors, and reduced-motion behavior.
3. Test supported viewport and assistive-technology combinations when available.
4. Record automated, manual, synthetic, and real-user evidence separately.
5. Escalate issues that require representative-user or qualified accessibility review.

## Required Output

Journey checklist, findings with severity, evidence class, screenshots or recordings where useful, and human testing gaps.

## Stop Conditions

- the artifact cannot be rendered or operated;
- a safety-critical journey lacks accessible fallback;
- automated checks are being treated as user evidence.

## Evaluation Cases

- accept a review that labels evidence class and covers keyboard/error paths;
- reject a pass based only on a linter when manual behavior was required.
