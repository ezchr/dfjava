# dfjava (moved)

Java Edition crossplay for Dragonfly now lives in the
[`crossplay` branch of ezchr/dragonfly](https://github.com/ezchr/dragonfly/tree/crossplay/server/java),
under `server/java`, next to the Dragonfly hooks it needs. It only ever worked with that branch, so
it is one place now: switch your server to the branch with one line in `go.mod` and import
`github.com/df-mc/dragonfly/server/java/javasession`. Setup is in
[server/java/README.md](https://github.com/ezchr/dragonfly/blob/crossplay/server/java/README.md).

The full history of this repository is kept in that branch. This repository is archived.
