Shaiya GX gx-v0.1.10

Changes since gx-v0.1.9:

• Complete mounts: two-second cast, original hand/appearance effects and sounds, jumping, backward walking, full lateral speed, one/two seats, invitations and passenger jump departure. Server rechecks boarding range; bones.ini groups every race.
• Restore the four original innate ITM weapon glows alongside enchantment effects, following the animated weapon correctly; fix weapon attachments and cloak shoulders.
• Complete native ladders, dungeon floor click movement, original dungeon minimaps and safer collision/landing checks. Repair rejected creature mesh headers offline; reduce cursor, collider and stationary-mob overhead.
• Complete goddess blessing UI and persistent obelisk/guard/boss/portal lifecycle. Restore spawn, recovery, combat and altar feedback; improve companion movement and stationary targeting.
• Add physical ranged projectiles; correct skill combos, reach/visibility checks, friendly self fallback, shared attack-speed recovery and movement pauses.
• Extend PvP to all 31 ranks and one million kills, original titles and Warrior Rewards. Correct guild captions; add GM money, kill and six base-stat setters.
• Polish native chat, acquired-item icons, portraits, cooldowns, pickup approach, map/news panels and coordinates. Status Mode stays empty, Stat shows armor affinity; Z/X counts use native placement.
• Restore fall damage at half the previous uncapped curve, with six safe units and authoritative, loss-free environmental deaths.

Update client + server together: protocol 0x11122209. Keep your runtime DB. Existing runtimes need the mount-definition migration; refreshed game data includes the new audio, minimaps and repaired models. See the upgrade guide:
https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.10-upgrade.md
