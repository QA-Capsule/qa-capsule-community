---
icon: material/web-clock
---

# Selenium

Selenium is a **driver library**, not a reporter. Pair it with a runner that exports JUnit XML.

| Stack | Runner | Upload param |
|-------|--------|--------------|
| Java + TestNG/JUnit | Maven Surefire | `JUnit` |
| Python | Pytest | `Pytest` |
| BDD | Cucumber + Selenium | `Cucumber` |
| Robot | SeleniumLibrary | `RobotFramework` |
| Mobile | Appium | `Appium` |

## Java (Maven + Surefire)

```xml
<dependency>
  <groupId>org.seleniumhq.selenium</groupId>
  <artifactId>selenium-java</artifactId>
  <version>4.20.0</version>
</dependency>
```

```bash
mvn test
curl -f -S ... -F "file=@target/surefire-reports/TEST-com.example.UITest.xml"
```

## Python (Pytest)

```bash
pip install pytest selenium
pytest ui/ --junitxml=pytest-results.xml
curl -f -S ... "?framework=Pytest" -F "file=@pytest-results.xml"
```

## Robot Framework + SeleniumLibrary

```bash
export SELENIUM_ENABLED=true
export SELENIUM_BROWSER=headlesschrome
./scripts/run-tests.sh
```

GitHub: `browser-actions/setup-chrome@v1` before UI suites.

← [All frameworks](../test-frameworks.md)
