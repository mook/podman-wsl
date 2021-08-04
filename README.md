# podman-wsl

This is a prepackaged version of [Podman] to run in a WSL2 container, using a
single executable.

This is based on Podman (both imported packages as well as copied code).

[Podman]: https://podman.io

## Installing

- Install [WSL2](https://aka.ms/wsl2-install).
- Run `podman-wsl.exe` and it will set things up as necessary.

## Usage

Run it as a replacement of the `podman` command:

```pwsh
.\podman-wsl.exe run --rm -t -i opensuse/leap:latest
```

It reads `$APPDATA\containers\containers.conf` for any configuration.
