# Tide-whisperer

Data access API for tidepool

# 0.6.0 - 2020-04-23
### Added 
- PT-1193 New API access point : compute time in range data for a set of users (last 24 hours)

# 0.5.2 - 2020-04-14
### Engineering
- PT-1232 Integrate latest changes from Tidepool develop branch
- PT-1034 Review API structure
- PT-1005 Openapi documentation

# 0.5.1 - 2020-03-26
### Fixed
- PT-1220 ReservoirChange objects are not retrieved

# 0.5.0 - 2020-03-19
### Changed
- PT-1150 Add filter on parameter level based on model

# 0.4.0 - 2019-10-28 
### Added 
- PT-734 Display the application version number on the status endpoint (/status).

# 0.3.2 
### Fixed 
- PT-649 Get Level 2 and 3 parameters for parameter history

## 0.3.1
### Added
- PT-607 DBLHU users should access to Level 1 and Level 2 parameters in the parameters history.

## 0.3.0
### Added
- PT-511 Access diabeloop system parameters history from tide-whisperer

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
