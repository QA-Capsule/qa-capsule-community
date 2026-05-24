---
icon: material/cellphone
---

# Appium (mobile)

| | |
|---|---|
| **Upload param** | `Appium` |
| **Report** | JUnit from TestNG/JUnit runner |

Appium tests usually run under **TestNG**, **JUnit**, **pytest**, or **WebdriverIO**.

## WebdriverIO + JUnit

```javascript
// wdio.conf.js
exports.config = {
  reporters: [
    'spec',
    ['junit', { outputDir: './results', outputFileFormat: () => 'wdio-junit.xml' }],
  ],
};
```

```bash
npx wdio run wdio.conf.js
curl -f -S ... "?framework=Appium" -F "file=@results/wdio-junit.xml"
```

## Java + TestNG + Appium

Use Surefire reports → `?framework=Appium` or `JUnit`.

← [All frameworks](../test-frameworks.md)
