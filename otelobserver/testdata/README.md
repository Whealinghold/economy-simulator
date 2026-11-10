# otelobserver testdata

- `catalogue_copy.md`: the tables of `doc/specs/spec-observability.md` 6.2.1 and
  6.2.2, from the line `#### 6.2.1` up to the line before `### 6.3`. The root test
  `catalogue_otel_copy_test.go` fails when it differs from the spec.
- `all_events.txt`: the type names of `event.go` that implement `Event`, sorted.
  The same root test fails when it differs from `event.go`.

When the spec table or `event.go` changes, copy the new text here in the same change.
