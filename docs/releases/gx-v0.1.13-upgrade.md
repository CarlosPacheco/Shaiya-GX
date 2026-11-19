# Updating to gx-v0.1.13

Update the client, GX Server, and loose client data together. Back up the server
runtime and install the new version in a separate folder if you need rollback.

1. Close the client and stop GX Server with the console command “shutdown server”.
2. Back up the server runtime and local configuration, including its SQLite DB,
   configuration, and private TLS material. Keep client settings and Studio
   backups separately.
3. Install the updated client, GX Studio, server binaries and resources, and
   matching loose client data. Keep your existing runtime and use the refreshed
   seed to start new test installations.
4. Start the server. GX applies schema migrations automatically on startup.
   Review startup output and check edited NPC, quest, map, enchant and teleport
   catalogs against your intended configuration.

GitHub's combined binary package includes fonts, Studio, and the server seed and
resources. Use the matching loose game data from the full maintainer bundle for
a complete gx-v0.1.13 installation. The existing Drive link contains v0.1.12.

[Release notes](gx-v0.1.13.md).
