# Tide-whisperer

Data access API for tidepool
# 0.3.2 
### Fixed 
- [PT-649] Get Level 2 and 3 parameters for parameter history

## 0.3.1
### Added
- [PT-607] DBLHU users should access to Level 1 and Level 2 parameters in the parameters history.

## 0.3.0
### Added
- [PT-511] Access diabeloop system parameters history from tide-whisperer

## 0.2.0 
### Added
- Integration from Tidepool latest changes

  Need to provide a new configuration item _auth_ in _TIDEPOOL_TIDE_WHISPERER_ENV_  (see [.vscode/launch.json.template](.vscode/launch.json.template) or [env.sh](env.sh) for example)

### Changed
- Update to MongoDb 3.6 drivers in order to use replica set connections. 

## dblp.0.1.2 - 2019-04-17

### Changed
- Fix status response of the service. On some cases (MongoDb restart mainly) the status was in error whereas all other entrypoints responded.

## dblp.0.1.1 - 2019-01-28

### Changed
- Remove dependency on lib-sasl

## dblp.0.1.0 - 2019-01-28

### Added
- Add support to MongoDb Authentication

## dblp.0.a - 2018-06-28

### Added
- Enable travis CI build 
