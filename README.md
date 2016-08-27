# sg-proto [![Join the chat at https://gitter.im/synapse-garden/sg-proto](https://badges.gitter.im/synapse-garden/sg-proto.svg)](https://gitter.im/synapse-garden/sg-proto?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) [![Windows build status](https://ci.appveyor.com/api/projects/status/hokjkj94b1vxx4nw/branch/master?svg=true)](https://ci.appveyor.com/project/binary132/sg-proto/branch/master) [![Linux build status](https://travis-ci.org/synapse-garden/sg-proto.svg?branch=master)](https://travis-ci.org/synapse-garden/sg-proto) [![Go Report Card](https://goreportcard.com/badge/github.com/synapse-garden/sg-proto)](https://goreportcard.com/report/github.com/synapse-garden/sg-proto)


```bash
go get github.com/synapse-garden/sg-proto
sg #              \
# -addr 127.0.0.1 \
# -port :12345    \
# -db my.db       \
# -cert cert.pem  \ # (To use SSL / HTTPS)
# -key cert.key   \
# -cfg conf.toml
```

## Config

The `conf.toml` file specifies config options.

## [TODO](TODO.md)

## [CHANGELOG](CHANGELOG.md)

### [LICENSE](LICENSE.txt)

sg-proto is licensed under the GNU Affero GPL v3.

It hosts its own source (when running) at /source, which satisfies the terms of the contract.
If you run a modified copy, please update rest/source.go to indicate the correct URL where
your modified source can be found.

Using sg-proto as a backend service does not cause your app to be "infected" by GPL.  However,
you do need to make sure you make the source code available (e.g. by reverse-proxying a public
link to /source and indicating that your software depends on this code.)

Other popular software licensed under Affero GPL includes:

 - [MongoDB](https://github.com/mongodb/mongo/blob/master/GNU-AGPL-3.0.txt)
 - [Diaspora](https://github.com/diaspora/diaspora/blob/develop/COPYRIGHT)
 - [Neo4j Enterprise](https://github.com/neo4j/neo4j/blob/3.1/enterprise/LICENSE.txt)
 - [Wikidot](https://github.com/gabrys/wikidot/blob/master/LICENSE.txt)

The purpose of this license is to ensure that SG code which the community benefits from
is released back to the community.  This license does not affect code managed or hosted
under this software, nor does it compel you to free-license any code which is not
statically linked to (i.e., a binary extension of) this code.
