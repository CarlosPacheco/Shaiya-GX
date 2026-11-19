"""Paths for the native client Item Mall catalog."""

from pathlib import Path

CASH_SDATA = Path('Item/Cash.SData')
LEGACY_CASH_SDATA = Path('Cash.SData')


def existing_cash_path(data: Path) -> Path:
    current = data / CASH_SDATA
    return current if current.exists() else data / LEGACY_CASH_SDATA
