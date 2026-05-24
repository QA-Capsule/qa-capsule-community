---
icon: material/language-python
---

# Pytest

| | |
|---|---|
| **Upload param** | `?framework=Pytest` |
| **Report** | `pytest-results.xml` |
| **Repo workflow** | `.github/workflows/api-tests-pytest.yml` |

## 1. JUnit output

```bash
pip install pytest
pytest tests/ -v --junitxml=pytest-results.xml
```

`pytest.ini`:

```ini
[pytest]
junit_family = xunit2
```

## 2. Upload

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Pytest" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -F "file=@pytest-results.xml"
```

## 3. GitHub Actions

```yaml
- run: pytest test_complex.py -v --junitxml=pytest-results.xml
  continue-on-error: true

- name: Upload to QA Capsule
  if: always()
  env:
    QA_CAPSULE_URL: ${{ secrets.QA_CAPSULE_URL }}
    QA_CAPSULE_API_KEY: ${{ secrets.QA_CAPSULE_API_PYTEST_KEY }}
  run: |
    curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Pytest" \
      -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -F "file=@pytest-results.xml"
```

## With Selenium

Use `pytest-selenium` or `selenium` + same `--junitxml`; upload still uses `?framework=Pytest`.

← [All frameworks](../test-frameworks.md)
