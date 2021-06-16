# Changelog

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
