---
icon: material/api
---

# Postman / Newman

| | |
|---|---|
| **Upload param** | `?framework=Postman` |
| **Report** | `newman-results.xml` |

## 1. Install & run

```bash
npm install -g newman
newman run MyCollection.postman_collection.json \
  -e staging.postman_environment.json \
  --reporters cli,junit \
  --reporter-junit-export newman-results.xml
```

## 2. Upload

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Postman" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -F "file=@newman-results.xml"
```

## 3. GitHub Actions

```yaml
- run: |
    npm install -g newman
    newman run collection.json --reporters junit --reporter-junit-export newman-results.xml
  continue-on-error: true

- name: Upload to QA Capsule
  if: always()
  run: |
    curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Postman" \
      -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -F "file=@newman-results.xml"
```

← [All frameworks](../test-frameworks.md)
