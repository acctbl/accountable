# Dev container

Open with the Dev Containers extension or **Code → Create codespace**.
First create runs the same locked `task setup` as CI.
Then use the root tasks (`task check`, `task ci`, `task ci:security`).

No host credentials, databases, or Docker sockets are mounted.
Tool and package versions come only from the checked-in lockfiles.
Playwright OS libraries are in the image; browser binaries install at `task setup`.

Codespaces prebuilds are off on purpose.
The parity workflow rebuilds this image and runs `task ci` on every PR and push to `main`.
