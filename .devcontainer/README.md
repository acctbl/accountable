# Dev container

Open the repository with the Dev Containers extension or **Code → Create
codespace**. The first create installs the locked toolchain and runs the same
`task setup` command used by CI; then use the normal root commands, for
example:

```sh
task check
task ci
task ci:security
```

The definition does not mount host credentials, databases, or Docker sockets.
It uses only the repository's checked-in tool and package lockfiles, and the
test suite uses its safe local fixtures. Playwright browser OS libraries are
baked into the image; Chromium, Firefox, and WebKit binaries are installed
during `task setup`, not baked into the image.

Codespaces prebuilds are intentionally not configured. The parity workflow
builds this definition from scratch and runs `task ci` inside it on every pull
request and push to `main`.
