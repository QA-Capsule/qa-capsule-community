---
icon: material/lightning-bolt
---

# Karate (API testing)

| | |
|---|---|
| **Upload param** | `Karate` |
| **Report** | `target/karate-reports/karate-summary.xml` |

## Maven

```xml
<dependency>
  <groupId>com.intuit.karate</groupId>
  <artifactId>karate-junit5</artifactId>
  <version>1.4.1</version>
  <scope>test</scope>
</dependency>
```

```bash
mvn test -Dtest=TestRunner
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Karate" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -F "file=@target/karate-reports/karate-summary.xml"
```

Karate emits Cucumber-compatible JUnit XML under `target/karate-reports/`.

← [All frameworks](../test-frameworks.md)
