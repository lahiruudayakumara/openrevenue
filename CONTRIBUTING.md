# Contributing

Open an issue in the [OpenCorex OpenRevenue repository](https://github.com/opencorex-org/openrevenue/issues) before substantial design work. Start with the [reproducible developer setup](docs/development/getting-started.md), then run `make bootstrap` and `make quality` before opening a pull request.

Keep bounded contexts isolated, add forward-only migrations, preserve rule/form versions, use minor units for money, and add tests. Never use production credentials, taxpayer exports, or real personal data in code, examples, fixtures, logs, screenshots, or issues. Commits should follow Conventional Commits. Contributions require the Developer Certificate of Origin (`Signed-off-by`). See the architecture and development guides before submitting a pull request.
