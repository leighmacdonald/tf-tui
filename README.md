# tf-tui

## Ensure launch options

Add the following launch options:
`+con_timestamp 1 -rpt -g15 -usercon +ip 0.0.0.0 +sv_rcon_whitelist_address 127.0.0.1 +rcon_password tf-tui`

## Debug log

If you set `DEBUG=1` env var, a log file will be created for extra error logging & debug messages.

Linux: `~/.config/tf-tui/tf-tui.log`

25.50


