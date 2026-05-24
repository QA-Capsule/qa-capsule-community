---
icon: material/nodejs
---

# Jest / Mocha / Vitest

| Runner | JUnit plugin | Upload param |
|--------|--------------|--------------|
| **Jest** | `jest-junit` | `Jest` |
| **Mocha** | `mocha-junit-reporter` | `Mocha` |
| **Vitest** | built-in `junit` reporter | `Vitest` |

## Jest

```bash
npm install -D jest jest-junit
```

```javascript
// jest.config.js
module.exports = {
  reporters: [
    'default',
    ['jest-junit', { outputDirectory: '.', outputName: 'jest-junit.xml' }],
  ],
};
```

```bash
npm test
curl -f -S ... "?framework=Jest" -F "file=@jest-junit.xml"
```

## Mocha

```javascript
// .mocharc.json
{
  "reporter": "mocha-junit-reporter",
  "reporterOptions": { "mochaFile": "mocha-junit.xml" }
}
```

## Vitest

```typescript
// vitest.config.ts
export default {
  reporters: ['default', 'junit'],
  outputFile: { junit: 'vitest-junit.xml' },
};
```

← [All frameworks](../test-frameworks.md)
