---
icon: material/format-list-checks
---

# TestNG

| | |
|---|---|
| **Upload param** | `?framework=JUnit` or `TestNG` |
| **Report** | `target/surefire-reports/testng-results.xml` |

## Maven

```xml
<plugin>
  <groupId>org.apache.maven.plugins</groupId>
  <artifactId>maven-surefire-plugin</artifactId>
  <configuration>
    <suiteXmlFiles>
      <suiteXmlFile>testng.xml</suiteXmlFile>
    </suiteXmlFiles>
  </configuration>
</plugin>
```

```bash
mvn test
curl -f -S ... "?framework=TestNG" \
  -F "file=@target/surefire-reports/testng-results.xml"
```

## testng.xml

```xml
<!DOCTYPE suite SYSTEM "https://testng.org/testng-1.0.dtd">
<suite name="API Suite">
  <test name="Smoke">
    <classes>
      <class name="com.example.SmokeTest"/>
    </classes>
  </test>
</suite>
```

← [All frameworks](../test-frameworks.md)
