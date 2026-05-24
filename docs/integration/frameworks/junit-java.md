---
icon: material/language-java
---

# JUnit 5 / Maven / Gradle (Java)

| | |
|---|---|
| **Upload param** | `?framework=JUnit` |
| **Report** | `target/surefire-reports/TEST-*.xml` |

## Maven Surefire

```xml
<plugin>
  <groupId>org.apache.maven.plugins</groupId>
  <artifactId>maven-surefire-plugin</artifactId>
  <version>3.2.5</version>
  <configuration>
    <reportsDirectory>${project.build.directory}/surefire-reports</reportsDirectory>
  </configuration>
</plugin>
```

```bash
mvn clean test
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=JUnit" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${BUILD_NUMBER}" \
  -F "file=@target/surefire-reports/TEST-com.example.MyTest.xml"
```

## Gradle

```kotlin
tasks.test {
    useJUnitPlatform()
    reports.junitXml.outputLocation.set(layout.buildDirectory.dir("test-results"))
}
```

Upload `build/test-results/test/TEST-*.xml`.

## Multi-module

Merge XML with `junit-report-merger` or upload each module file in a loop.

← [All frameworks](../test-frameworks.md)
