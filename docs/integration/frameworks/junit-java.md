---
icon: material/language-java
---

# JUnit 5 (Java)

| | |
|---|---|
| **Upload param** | `?framework=JUnit` |
| **Report** | `reports/TEST-junit-jupiter.xml` |
| **Repo workflow** | `.github/workflows/api-tests-junit-java.yml` |
| **Test folder** | `tests/junit-java/src/` |
| **Secret** | `QA_CAPSULE_API_JUNIT_JAVA_KEY` |

## Test suites in this repository

Uses `junit-platform-console-standalone` — no Maven or Gradle required.

| Class | Tests | Expected result |
|---|---|---|
| `UserServiceTest` | create user 201, get user 200, delete 404 | 2 pass · 1 fail |
| `PaymentServiceTest` | charge valid card, charge expired card, refund | 2 pass · 1 fail |
| `InventoryServiceTest` | stock check, reserve stock, out-of-stock exception | 2 pass · 1 error |

## 1. Run with JUnit Platform Console

```bash
# Download once
curl -sL https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/1.10.2/junit-platform-console-standalone-1.10.2.jar \
  -o junit-standalone.jar

# Compile
javac -cp junit-standalone.jar tests/junit-java/src/*.java -d classes/

# Run
java -jar junit-standalone.jar \
  --scan-classpath \
  --class-path classes/ \
  --reports-dir reports/
```

## 2. Run with Maven Surefire

```xml
<plugin>
  <groupId>org.apache.maven.plugins</groupId>
  <artifactId>maven-surefire-plugin</artifactId>
  <version>3.2.5</version>
</plugin>
```

```bash
mvn clean test
```

Report: `target/surefire-reports/TEST-*.xml`

## 3. Upload to QA Capsule

```bash
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=JUnit" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@reports/TEST-junit-jupiter.xml"
```

## 4. GitHub Actions

```yaml
- name: Download JUnit Platform Console
  run: |
    curl -sL https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/1.10.2/junit-platform-console-standalone-1.10.2.jar \
      -o junit-standalone.jar

- name: Compile Tests
  run: javac -cp junit-standalone.jar tests/junit-java/src/*.java -d classes/

- name: Run JUnit Tests
  run: |
    java -jar junit-standalone.jar \
      --scan-classpath \
      --class-path classes/ \
      --reports-dir reports/
  continue-on-error: true

- name: Send Alert to QA Capsule
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_JUNIT_JAVA_KEY }}
  run: |
    curl -X POST "$WEBHOOK_URL/api/webhooks/upload?framework=JUnit" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Execution-Env: STAGING" \
      -H "X-Execution-Type: TEST-RUN" \
      -F "file=@reports/TEST-junit-jupiter.xml"
```

!!! note "Headers"
    - `X-Run-Id` groups all test results under the same pipeline run in the Execution Hub.
    - `X-Execution-Env` accepts `PROD`, `STAGING`, `INTEGRATION`, `DEV`.
    - `X-Execution-Type` accepts `TEST-RUN`, `NIGHTLY`, `SMOKE`, `REAL`.

## Gradle

```kotlin
tasks.test {
    useJUnitPlatform()
    reports.junitXml.outputLocation.set(layout.buildDirectory.dir("test-results"))
}
```

Upload `build/test-results/test/TEST-*.xml`.

← [All frameworks](../test-frameworks.md)
