---
icon: material/shield-check
---

# REST Assured (Java API)

| | |
|---|---|
| **Upload param** | `JUnit` |
| **Report** | Maven Surefire XML |

REST Assured tests run as JUnit/TestNG classes:

```java
@Test
public void getUserReturns200() {
    given().when().get("/users/1").then().statusCode(200);
}
```

```bash
mvn test
curl -f -S ... "?framework=JUnit" \
  -F "file=@target/surefire-reports/TEST-com.example.ApiTest.xml"
```

← [All frameworks](../test-frameworks.md)
