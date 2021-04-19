<!--
parent:
  order: false
-->


# Cosmos SDK




<div align="center">
  <a href="https://github.com/cosmos/cosmos-sdk/releases/latest">
    <img alt="Version" src="https://img.shields.io/github/tag/cosmos/cosmos-sdk.svg" />
  </a>
  <a href="https://github.com/cosmos/cosmos-sdk/blob/master/LICENSE">
    <img alt="License: Apache-2.0" src="https://img.shields.io/github/license/cosmos/cosmos-sdk.svg" />
  </a>
  <a href="https://pkg.go.dev/github.com/allinbits/cosmos-sdk?tab=doc">
    <img alt="GoDoc" src="https://godoc.org/github.com/allinbits/cosmos-sdk?status.svg" />
  </a>
  <a href="https://goreportcard.com/report/github.com/allinbits/cosmos-sdk">
    <img alt="Go report card" src="https://goreportcard.com/badge/github.com/allinbits/cosmos-sdk" />
  </a>
  <a href="https://codecov.io/gh/allinbits/cosmos-sdk">
    <img alt="Code Coverage" src="https://codecov.io/gh/allinbits/cosmos-sdk/branch/master/graph/badge.svg" />
  </a>
</div>
<div align="center">
  <a href="https://github.com/allinbits/cosmos-sdk">
    <img alt="Lines Of Code" src="https://tokei.rs/b1/github/allinbits/cosmos-sdk" />
  </a>
  <a href="https://discord.gg/AzefAFd">
    <img alt="Discord" src="https://img.shields.io/discord/669268347736686612.svg" />
  </a>
  <a href="https://sourcegraph.com/github.com/allinbits/cosmos-sdk?badge">
    <img alt="Imported by" src="https://sourcegraph.com/github.com/allinbits/cosmos-sdk/-/badge.svg" />
  </a>
    <img alt="Sims" src="https://github.com/allinbits/cosmos-sdk/workflows/Sims/badge.svg" />
    <img alt="Lint Satus" src="https://github.com/allinbits/cosmos-sdk/workflows/Lint/badge.svg" />
</div>

The Cosmos-SDK is a framework for building sovereign, interconnected blockchain applications in Golang.
It is being used to build [`Gaia`](https://github.com/cosmos/gaia), the first implementation of the Cosmos Hub.

**Note**: Requires [Go 1.16+](https://golang.org/dl/)

## Quick Start

To learn how the SDK works from a high-level perspective, go to the [SDK Intro](./docs/intro/overview.md).

If you want to get started quickly and learn how to build on top of the SDK, please follow the [SDK Application Tutorial](https://tutorials.cosmos.network/nameservice/tutorial/00-intro.html). You can also fork the tutorial's repository to get started building your own Cosmos SDK application.

For more, please go to the [Cosmos SDK Docs](./docs/).

## Interblockchain Communication (IBC)

The IBC module for the SDK has moved to its [own repository](https://github.com/cosmos/ibc-go). Go there to build and integrate with the IBC module. 

## Starport

If you are starting a new app or a new module you can use [Starport](https://github.com/tendermint/starport) to help you get started and speed up development. If you have any questions or find a bug, feel free to open an issue in the repo.
