# tide-whisperer

[![Build Status](https://travis-ci.com/tidepool-org/tide-whisperer.png)](https://travis-ci.com/tidepool-org/tide-whisperer)

Data access API for tidepool

## Package dependencies

This repository/project makes use of [dep](https://github.com/golang/dep) to handle dependencies.
_vendor_ folder is not pushed in GH repository. It is re-generated through the command
```shell
$GOPATH/bin/dep ensure
```

This command is also run in the Travis configuration so the dependencies folder is built and present for the project build.