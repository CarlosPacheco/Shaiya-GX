# Updating to gx-v0.1.14

1. Stop GX Server with `shutdown server` and close the client and GX Studio.
2. Back up your SQLite runtime, local configuration and edited catalogs. Keep these backups separate from the new installation.
3. Install the matching gx-v0.1.14 client, GX Studio, server binaries, server resources and loose client data in a writable directory. Carry over your existing runtime and review local catalog changes before merging them into the new defaults.
4. Start the terminal console and enter `start server`. Existing runtimes migrate automatically. A fresh installation copies the included schema-31 development seed.

The GitHub archives include the external fonts, vehicle bindings and full server resources. The [complete gx-v0.1.14 portable on Google Drive](https://drive.google.com/file/d/1TPyHqlapcyItIbuVFsL5ISzapfQPbnPC/view?usp=sharing) also includes the matching loose client data with the updated skill and quest catalogs and personal market art.

[Release notes](gx-v0.1.14.md) · [Portable layout and requirements](../client-portable.md)
