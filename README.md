# dfjava (moved)

Java Edition crossplay for Dragonfly now lives in the
[`java-native` branch of ezchr/dragonfly](https://github.com/ezchr/dragonfly/tree/java-native/server/java),
under `server/java`, next to the Dragonfly hooks it needs. It only ever worked with that branch, so
it is one place now: switch your server to the branch with one line in `go.mod` and import
`github.com/df-mc/dragonfly/server/java/javasession`. Setup is in
[server/java/README.md](https://github.com/ezchr/dragonfly/blob/java-native/server/java/README.md).

The full history of this repository is kept in that branch. This repository is archived.
