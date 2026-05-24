---
icon: material/language-ruby
---

# RSpec (Ruby)

| | |
|---|---|
| **Upload param** | `RSpec` |
| **Report** | `rspec.xml` |

## Gemfile

```ruby
gem 'rspec'
gem 'rspec_junit_formatter'
```

```ruby
# .rspec
--format RspecJunitFormatter
--out rspec.xml
--format documentation
```

```bash
bundle exec rspec
curl -f -S ... "?framework=RSpec" -F "file=@rspec.xml"
```

← [All frameworks](../test-frameworks.md)
