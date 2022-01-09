# Changelog

## [4.10.0](https://www.github.com/contiamo/go-base/compare/v4.9.0...v4.10.0) (2022-01-09)


### Features

* add option for trimming SQL in open tracing spans ([#192](https://www.github.com/contiamo/go-base/issues/192)) ([8347610](https://www.github.com/contiamo/go-base/commit/83476100d5f9698a23a71ab46d017ab61dcd0fa5))

## [4.9.0](https://www.github.com/contiamo/go-base/compare/v4.8.0...v4.9.0) (2021-10-25)


### Features

* support PGSSL* env variables for configuring the ssl mode ([#187](https://www.github.com/contiamo/go-base/issues/187)) ([96447ad](https://www.github.com/contiamo/go-base/commit/96447adbea23e00d0ea691f5bf84b88b1d2a86c3))

## [4.8.0](https://www.github.com/contiamo/go-base/compare/v4.7.0...v4.8.0) (2021-10-19)


### Features

* **CON-4069:** add in-memory cache for JWT public keys ([#184](https://www.github.com/contiamo/go-base/issues/184)) ([db1a120](https://www.github.com/contiamo/go-base/commit/db1a12052fee0b3ff1118ac694b9b49b0d097b60))

## [4.7.0](https://www.github.com/contiamo/go-base/compare/v4.6.0...v4.7.0) (2021-10-15)


### Features

* check progress for status:error and return error if found. ([#182](https://www.github.com/contiamo/go-base/issues/182)) ([fc4d29f](https://www.github.com/contiamo/go-base/commit/fc4d29fb207aeddd3142b9d375a489b8180d8a47))

## [4.6.0](https://www.github.com/contiamo/go-base/compare/v4.5.2...v4.6.0) (2021-10-08)


### Features

* implement heartbeat timeout in the worker ([#167](https://www.github.com/contiamo/go-base/issues/167)) ([6a97dbf](https://www.github.com/contiamo/go-base/commit/6a97dbf698914b1c05752e001b135a8e9fcf20a4))

### [4.5.2](https://www.github.com/contiamo/go-base/compare/v4.5.1...v4.5.2) (2021-10-01)


### Bug Fixes

* Revert "feat: use pg_advisory_lock to control db migration concurrency ([#165](https://www.github.com/contiamo/go-base/issues/165))" ([#176](https://www.github.com/contiamo/go-base/issues/176)) ([e79943c](https://www.github.com/contiamo/go-base/commit/e79943ce8e76799c0101300b454944605b08890c))

### [4.5.1](https://www.github.com/contiamo/go-base/compare/v4.4.1...v4.5.1) (2021-09-30)


### Features

* remove goserver as dependency ([#174](https://www.github.com/contiamo/go-base/issues/174)) ([67b1950](https://www.github.com/contiamo/go-base/commit/67b19504057b948c701d931a8a3a29da79961213))
* update `contiamo/jwt` to fix CVE ([#170](https://www.github.com/contiamo/go-base/issues/170)) ([1a58be6](https://www.github.com/contiamo/go-base/commit/1a58be6dec199197811d4cc6a981e745cfc2828d))
* update go-server to v0.6.0 ([#172](https://www.github.com/contiamo/go-base/issues/172)) ([9b674c4](https://www.github.com/contiamo/go-base/commit/9b674c456f309d3623a8f60e074583275402bbd2))
* use pg_advisory_lock to control db migration concurrency ([#165](https://www.github.com/contiamo/go-base/issues/165)) ([fcea24b](https://www.github.com/contiamo/go-base/commit/fcea24bd94f459af6bbb6a3997ce5e1a6ebdf6bc))


### Bug Fixes

* go.sum ([#173](https://www.github.com/contiamo/go-base/issues/173)) ([a82b3eb](https://www.github.com/contiamo/go-base/commit/a82b3eb4b62ab79a88aa34dff7b7d4b087e8b521))


### Miscellaneous Chores

* release 4.5.1 ([71456a8](https://www.github.com/contiamo/go-base/commit/71456a8751986d8c99310ac7b629500218365a04))

### [4.4.1](https://www.github.com/contiamo/go-base/compare/v4.4.0...v4.4.1) (2021-09-10)


### Bug Fixes

* do not allow infinite retry by default in the base api client ([#163](https://www.github.com/contiamo/go-base/issues/163)) ([1980674](https://www.github.com/contiamo/go-base/commit/19806746d1bfe0bf57b3952d82702e4f1b87a9c0))

## [4.4.0](https://www.github.com/contiamo/go-base/compare/v4.3.0...v4.4.0) (2021-08-27)


### Features

* Add WithRetry to the BaseAPIClient interface ([#155](https://www.github.com/contiamo/go-base/issues/155)) ([7f77db1](https://www.github.com/contiamo/go-base/commit/7f77db1fe70b0225661574ace50dde2a98c0b96e))


### Bug Fixes

* treat json api task body as json ([#159](https://www.github.com/contiamo/go-base/issues/159)) ([5fe641f](https://www.github.com/contiamo/go-base/commit/5fe641f9da0169d4c3d9d974684f5661143b02f2))

## [4.3.0](https://www.github.com/contiamo/go-base/compare/v4.2.1...v4.3.0) (2021-08-02)


### Features

* add support for setting headers in API client ([#151](https://www.github.com/contiamo/go-base/issues/151)) ([ab297d0](https://www.github.com/contiamo/go-base/commit/ab297d0a92bae67bdb80692d68915ed0f4fb363e))

### [4.2.1](https://www.github.com/contiamo/go-base/compare/v4.2.0...v4.2.1) (2021-07-26)


### Bug Fixes

* upgrade JWT library ([#149](https://www.github.com/contiamo/go-base/issues/149)) ([29b4dbc](https://www.github.com/contiamo/go-base/commit/29b4dbcf5a5ce8a33d4a43516a5c13158b8acc20))

## [4.2.0](https://www.github.com/contiamo/go-base/compare/v4.1.0...v4.2.0) (2021-06-17)


### Features

* Allow cloning API Clients with new token providers ([#144](https://www.github.com/contiamo/go-base/issues/144)) ([da0adce](https://www.github.com/contiamo/go-base/commit/da0adce8189f83d8a5e9e60ce4520280bcd40474))

## [4.1.0](https://www.github.com/contiamo/go-base/compare/v4.0.0...v4.1.0) (2021-06-17)


### Features

* Add request method to BaseAPIClient that expose the response object ([#142](https://www.github.com/contiamo/go-base/issues/142)) ([a1e93eb](https://www.github.com/contiamo/go-base/commit/a1e93eb7d105fa9983f1cb9c6e9e28ae49349fd8))

## [4.0.0](https://www.github.com/contiamo/go-base/compare/v3.6.1...v4.0.0) (2021-06-16)


### ⚠ BREAKING CHANGES

* remove generators package (#140)

### Features

* check that ids exist under a filter when resolving. ([a2c1173](https://www.github.com/contiamo/go-base/commit/a2c1173ca9bade6efbf9fc4806369bf84d75db08))
* **CON-3568:** check that ids exist under a filter when resolving. ([#136](https://www.github.com/contiamo/go-base/issues/136)) ([a2c1173](https://www.github.com/contiamo/go-base/commit/a2c1173ca9bade6efbf9fc4806369bf84d75db08))


### Bug Fixes

* sort field errors not only by message, but by key (if any) and then by message ([#138](https://www.github.com/contiamo/go-base/issues/138)) ([515dc65](https://www.github.com/contiamo/go-base/commit/515dc65a91d015cc0a71c30f893e16a39b451568))


### Miscellaneous Chores

* remove generators package ([#140](https://www.github.com/contiamo/go-base/issues/140)) ([d7a47a1](https://www.github.com/contiamo/go-base/commit/d7a47a1dab7e58d44a57f0c8d1dd1d4ea5b4e6ca))

### [3.6.1](https://www.github.com/contiamo/go-base/compare/v3.6.0...v3.6.1) (2021-05-31)


### Bug Fixes

* do not parse body if ouput dest is nil ([#135](https://www.github.com/contiamo/go-base/issues/135)) ([d001321](https://www.github.com/contiamo/go-base/commit/d001321d4ec967af97bf27843bff462c2474d7ca))
* make API errors more informative, no empty strings ([#130](https://www.github.com/contiamo/go-base/issues/130)) ([1b7e53d](https://www.github.com/contiamo/go-base/commit/1b7e53d5678a2211a1b1b7453726948243f05179))

## [3.6.0](https://www.github.com/contiamo/go-base/compare/v3.5.0...v3.6.0) (2021-05-19)


### Features

* Add error handling for Unsupported Media Type ([#126](https://www.github.com/contiamo/go-base/issues/126)) ([a6670c6](https://www.github.com/contiamo/go-base/commit/a6670c638c67c35327b9c214b1faeccc52b4061d))

## [3.5.0](https://www.github.com/contiamo/go-base/compare/v3.4.1...v3.5.0) (2021-05-11)


### Features

* add monitoring server setup utility ([49a4344](https://www.github.com/contiamo/go-base/commit/49a4344b3e00186442e82b2972e46cb36df9589d))
* Set logger context in logging middleware ([#120](https://www.github.com/contiamo/go-base/issues/120)) ([49a4344](https://www.github.com/contiamo/go-base/commit/49a4344b3e00186442e82b2972e46cb36df9589d))
