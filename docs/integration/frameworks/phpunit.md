---
icon: material/language-php
---

# PHPUnit

| | |
|---|---|
| **Upload param** | `PHPUnit` |
| **Report** | `build/logs/junit.xml` |

## phpunit.xml

```xml
<logging>
  <junit outputFile="build/logs/junit.xml"/>
</logging>
```

```bash
vendor/bin/phpunit
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=PHPUnit" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -F "file=@build/logs/junit.xml"
```

← [All frameworks](../test-frameworks.md)
