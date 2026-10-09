# smartctltui-go

A modern, polished TUI for viewing S.M.A.R.T. information about disks in Linux.

A wrapper around `smartctl` (Smartmontools) with a modern interface built on
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[lipgloss](https://github.com/charmbracelet/lipgloss).

## Features

- Automatic discovery of all disks (SATA, NVMe) via `smartctl --scan`.
- A per-disk card: model, serial number, capacity, firmware, type.
- Health (overall health), temperature, and endurance for NVMe.
- A SMART-attribute table with severity highlighting (green / yellow / red).
- Navigate between disks with arrow keys, PgUp/PgDn, R for manual refresh,
  Enter to change the auto-refresh interval, Q/Ctrl+C to quit.
- Auto re-launch under `sudo` if started without root (required for disk access).
- `--mock` — demo mode with sample data, no root needed.

## Requirements

- Go 1.27+
- `smartctl` from the `smartmontools` package (in most distros: `apt install smartmontools`, `dnf install smartmontools`, etc.)

## Installation

```bash
cd smartctltui-go
go install ./cmd/smartctltui
```

Or simply:

```bash
go run ./cmd/smartctltui
```

## Usage

```bash
# normal run (auto-sudo if not root)
smartctltui

# show the device list
smartctltui --list

# demo mode (sample data, no root)
smartctltui --mock
```

## Controls

| Key            | Action                     |
| -------------- | -------------------------- |
| `up` / `PgUp`  | previous disk              |
| `down` / `PgDn`| next disk                  |
| `R`            | refresh now                |
| `Enter`        | change interval (10→5→30 s) |
| `Q` / `Ctrl+C` | quit                       |

## Project structure

- `cmd/smartctltui/main.go` — entry point, flags.
- `internal/smart/smartctl.go` — `smartctl` output parser, data models, mock.
- `internal/ui/ui.go` — model, update, TUI rendering.

## License

MIT
