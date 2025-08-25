# tf-tui

[![Demo](docs/demo.webm)](docs/demo.webm)

## Ensure launch options

Add the following launch options:
`+con_timestamp 1 -rpt -g15 -usercon +ip 0.0.0.0 +sv_rcon_whitelist_address 127.0.0.1 +rcon_password tf-tui`


## Config file

There is a config editor in the app, however its currently *very* limited and only supports a couple fields. You can
access that with the `shift+e` shortcut.

Linux: `~/.config/tf-tui/tf-tui.yaml`

Windows: `%LOCALAPPDATA%\tf-tui\tf-tui.yaml`

```yaml
# Your own steamid
steam_id: "76561197970000000"

# rcon server config
address: 127.0.0.1:27015
password: tf-tui

# Path to your console.log
console_log_path: /home/<username>/.steam/steam/steamapps/common/Team Fortress 2/tf/console.log

# API URL
api_base_url: https://tf-api.roto.lol/

# Set of custom bot detector lists
# Doesn't currently use the data, but it will eventually.
bd_lists:
  - url: https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/refs/heads/master/staging/cfg/playerlist.official.json
    name: poozer

# Custom URL links to show
# The %s is replaced with the steamid of the format set. Empty defaults to steam64.
# format can be one of: steam64, steam, steam3
links:
  - url: https://demos.tf/profiles/%s
    name: demos.tf
    format: "steam64"
```

## Debug log

If you set `DEBUG=1` env var, a log file will be created for extra error logging & debug messages.

Linux: `~/.config/tf-tui/tf-tui.log` Tail shortcut: `make tail`

## Building

The only build dependency is `go1.24+`.

Full snapshot build using goreleaser. Binaries are output to the platform specific paths: `./dist/tf-tui_{linux,windows}/tf-tui`:

```sh
$ make snapshot
```

The standard go build options will also work, but some values will not be automatically injected into the build such as version/commit info.

```sh
$ go build -o tf-tui && ./tf-tui
```

### Releases

Releases are built and uploaded automatically when a new matching tag is pushed.

```sh
$ git tag -a v1.2.3 -m "v1.2.3 release!" && git push --tags
```
