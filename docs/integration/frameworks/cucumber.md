---
icon: material/carrot
---

# Cucumber (BDD)

| Stack | Upload param | Report |
|-------|--------------|--------|
| Java | `Cucumber` | `target/cucumber-reports/Cucumber.xml` |
| JavaScript | `Cucumber` | `cucumber-report.xml` |

## Java (Maven)

```xml
<plugin>
  <groupId>net.masterthought</groupId>
  <artifactId>maven-cucumber-reporting</artifactId>
  <executions>
    <execution>
      <id>cucumber-report</id>
      <phase>verify</phase>
      <goals><goal>generate</goal></goals>
    </execution>
  </executions>
</plugin>
```

```bash
mvn verify
curl -f -S ... "?framework=Cucumber" \
  -F "file=@target/cucumber-reports/Cucumber.xml"
```

## JavaScript

```javascript
// cucumber.js
module.exports = {
  format: ['junit:./reports/cucumber.xml'],
};
```

← [All frameworks](../test-frameworks.md)
