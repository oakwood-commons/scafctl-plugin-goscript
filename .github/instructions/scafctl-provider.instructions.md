---
description: "scafctl provider guidance: method responsibilities, schema design, WhatIf parity, configuration, and tests."
applyTo: "**/*.go"
---

# scafctl Provider Guidance

Use this guidance when editing provider plugin code.

## Method Responsibilities

Keep the provider lifecycle methods sharply separated.

- `GetProviderDescriptor` describes the provider contract.
- `ExecuteProvider` performs the real work.
- `DescribeWhatIf` explains what execution would do without side effects.
- `ConfigureProvider` stores host configuration for later use.
- `ExecuteProviderStream` should only be implemented when streaming output is truly supported.
- `ExtractDependencies` should only return resolver dependencies when provider inputs reference them.
- `StopProvider` should be safe and idempotent.

## Descriptor Rules

The descriptor is the source of truth for the provider contract.

- Keep `Name`, `DisplayName`, `Description`, `Capabilities`, and `Schema` consistent.
- Do not advertise capabilities that the implementation does not support.
- Ensure required schema fields match runtime expectations.
- Keep examples realistic and aligned with README snippets and tests.

## Execution Rules

- Reject unknown provider names consistently across all methods.
- Handle nil, empty, and zero-value inputs deliberately.
- Return structured output with stable field names.
- Avoid hidden global state and cross-call mutation.
- Wrap non-trivial errors with context.

## WhatIf Rules

`DescribeWhatIf` should be useful and trustworthy.

- Describe the same operation that `ExecuteProvider` would perform.
- Do not perform I/O, mutation, or other side effects.
- Mention important inputs when they affect behavior.
- Keep wording specific enough to help a reviewer verify intent.

## Configuration Rules

- Read host-provided settings through `ConfigureProvider`.
- Keep configuration validation close to where values are consumed.
- Do not hardcode secrets, tokens, URLs, or file system assumptions.

## Test Expectations

Every provider change should consider tests for:

- known provider vs unknown provider
- happy path execution
- invalid, nil, or empty input
- descriptor validity
- `DescribeWhatIf` parity with execution intent
- configuration-dependent behavior
- any new edge case introduced by the change