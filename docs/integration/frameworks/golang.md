---
icon: material/language-go
---

# Go (`testing` package)

| | |
|---|---|
| **Upload param** | `Go` |
| **Report** | `report.xml` |

## gotestsum (recommended)

```bash
go install gotest.tools/gotestsum@latest
gotestsum --junitfile report.xml ./...
```

## Upload

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Go" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${GITHUB_RUN_ID}" \
  -F "file=@report.xml"
```

## GitHub Actions

```yaml
- run: go install gotest.tools/gotestsum@latest && gotestsum --junitfile report.xml ./...
  continue-on-error: true

- name: Upload to QA Capsule
  if: always()
  run: |
    curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Go" \
      -H "X-API-Key: ${{ secrets.QA_CAPSULE_API_KEY }}" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -F "file=@report.xml"
```

← [All frameworks](../test-frameworks.md)
