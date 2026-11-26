# Upgrade to Shaiya GX gx-v0.1.22

1. Close the client and run `shutdown server` in GX Server's console. Back up
   your runtime SQLite database and local configuration.
2. Extract the new client and server packages into a fresh writable directory.
   Reuse compatible loose client data and preserve the supplied fonts and
   licenses. Keep your runtime database separate from the development seed.
3. Apply your custom server settings to the new configuration. Preserve the
   updated faction rebirth entries for zones 65 and 66 in `Map.ini`.
4. Update launcher arguments and firewall rules if you use the default ports.
   GX now uses TCP 31000 for login and 31010 for World. Installations with
   explicit `RAPTURE_LOGIN_LISTEN` and `RAPTURE_WORLD_LISTEN` settings can keep
   those values and connect with matching `-login` and `-world-port` arguments.
5. Start GX Server, then open the updated client. If your data pack provides
   language overlays, select the language in Options and restart the client.

See the [server guide](../server.md) for custom network settings and the
[portable guide](../client-portable.md) for the data layout.
