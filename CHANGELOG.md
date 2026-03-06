# Changelog

## [0.0.4](https://github.com/eleboucher/mktxp/compare/0.0.3...0.0.4) (2026-03-06)


### Features

* apply overengineer optimization for memory and cpu ([19b87f5](https://github.com/eleboucher/mktxp/commit/19b87f549771284c8784038527ca74367b40273f))
* properly handle bandwidth ([4df1a68](https://github.com/eleboucher/mktxp/commit/4df1a6844b7a94de8ed3867e971b17e08c0c019c))
* remove duplicated metrics ([0ce2ed3](https://github.com/eleboucher/mktxp/commit/0ce2ed39380998a353e8313f00d7c02ee8ac4d97))
* use counter where it's accumulating data ([b70afb8](https://github.com/eleboucher/mktxp/commit/b70afb89ec063bfbf00d4af0a07bd4a164f77669))
* use header for scrape timeout ([b7b603d](https://github.com/eleboucher/mktxp/commit/b7b603daafddb8c001c0eec8041c96c51fccf313))
* use semaphore to handle concurrency ([98466ad](https://github.com/eleboucher/mktxp/commit/98466adfcd0c70272177f94a95402b2b15cbe370))


### Miscellaneous Chores

* delete unecessary config ([6899ba2](https://github.com/eleboucher/mktxp/commit/6899ba297ead4d6505114d1ec3347a4287a3d1f9))
* handle prometheus error better ([a3cef6a](https://github.com/eleboucher/mktxp/commit/a3cef6a2ed3c0b38cb16c9c3fa5d9ab363aa3bd1))
* improve bandwidth concurrency ([906ac3b](https://github.com/eleboucher/mktxp/commit/906ac3bb3254e3f57f40c55b9fabf4f2454acd6e))

## [0.0.3](https://github.com/eleboucher/mktxp/compare/0.0.2...0.0.3) (2026-03-05)


### Features

* merge health and hw_health ([87e7644](https://github.com/eleboucher/mktxp/commit/87e764476bd4bf466294727cdbc783407f9ae429))
* **speedtest:** use github.com/showwin/speedtest-go ([2af5808](https://github.com/eleboucher/mktxp/commit/2af5808e0cc87e246817523ebb87edcbc787c4c4))

## [0.0.2](https://github.com/eleboucher/mktxp/compare/0.0.1...0.0.2) (2026-03-05)


### Bug Fixes

* **ci:** fix build and release ci ([edb8b45](https://github.com/eleboucher/mktxp/commit/edb8b4545f0dd3bfe9f84e124d2af84daf6aaa4a))

## 0.0.1 (2026-03-05)


### Features

* add missing collectors ([76f86fd](https://github.com/eleboucher/mktxp/commit/76f86fd703acf1a25231c75672cbe1421b1742ed))
* add missing metrics ([c0a23dc](https://github.com/eleboucher/mktxp/commit/c0a23dcab191c58c29e6f8d4f422278e5be4f664))
* allow routers settings to be override by env var ([b41a3c1](https://github.com/eleboucher/mktxp/commit/b41a3c114257fcbd88e81bceedd1dec531677cfd))
* rewrite mktxp in golang ([523359e](https://github.com/eleboucher/mktxp/commit/523359ed958f1667434829148c31d27290c86ba5))


### Bug Fixes

* **ci:** update golangci lint version ([fd67825](https://github.com/eleboucher/mktxp/commit/fd67825e611d86da9de7ec3c932ca2fde88805d7))
* clean duplicated labels ([f91adf5](https://github.com/eleboucher/mktxp/commit/f91adf5ffe188eb10d45205584b00694de131208))
* fix lint ([2ed3673](https://github.com/eleboucher/mktxp/commit/2ed36734088d5d42ec827e3203f96955bfa03a1d))
* fix the tests ([e6a0e28](https://github.com/eleboucher/mktxp/commit/e6a0e2847a945604dcf9bed2c7d938a3197fc299))
* fix typo in metrics declaration ([84b0e30](https://github.com/eleboucher/mktxp/commit/84b0e30ab6bc3e051e435d890ec319361821cac9))
* rename label correctly ([001e632](https://github.com/eleboucher/mktxp/commit/001e632123fcfe0cf5f44602db8be284b180342a))


### Documentation

* update readme with the latest feature ([ed26716](https://github.com/eleboucher/mktxp/commit/ed267166aaafe7475bc575bfc6a914932def78b1))


### Miscellaneous Chores

* **errors:** correctly handle error ([0d99056](https://github.com/eleboucher/mktxp/commit/0d990566e50844d947d7791c99c128b9cd433060))
* improve logging in case of insecure connection ([098457f](https://github.com/eleboucher/mktxp/commit/098457f8ab81560d891fcfa46eeaecc72369b2ca))


### Code Refactoring

* refactor config ([6545b9f](https://github.com/eleboucher/mktxp/commit/6545b9f94a1cb9619d70e4edca90a7194d30b139))
* rewrite the collector to be clean ([5f947c8](https://github.com/eleboucher/mktxp/commit/5f947c86efb56a9a59ac9048664c4037a62c6d6e))
