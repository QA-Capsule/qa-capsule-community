---
icon: material/code-braces
---

# NUnit / xUnit / MSTest (.NET)

| Framework | Upload param | Report |
|-----------|--------------|--------|
| **NUnit** | `NUnit` | `TestResults.xml` |
| **xUnit** | `xUnit` | `TestResults.xml` |
| **MSTest** | `MSTest` | `*.trx` → convert to JUnit |

## dotnet test + TRX (Azure DevOps friendly)

```bash
dotnet test --logger "trx;LogFileName=results.trx"
```

Convert TRX → JUnit with [trx2junit](https://github.com/spekt/trx2junit) then upload.

## NUnit + JUnit XML

```bash
dotnet test --results-directory ./TestResults --logger "junit;LogFilePath=./TestResults/nunit-junit.xml"
```

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=NUnit" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -F "file=@TestResults/nunit-junit.xml"
```

## xUnit

```xml
<PackageReference Include="JunitXml.TestLogger" Version="3.1.12" />
```

```bash
dotnet test --logger "junit;LogFilePath=./xunit-results.xml"
```

← [All frameworks](../test-frameworks.md)
