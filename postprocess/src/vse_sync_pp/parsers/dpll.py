### SPDX-License-Identifier: GPL-2.0-or-later

"""Parse dpll log messages"""

from collections import namedtuple

from .parser import (Parser, parse_timestamp, parse_decimal)


def _int_state(val, default=-1):
    """Parse DPLL state field; empty or missing values map to unknown (-1)."""
    if val is None or val == '':
        return default
    return int(val)


class TimeErrorParser(Parser):
    """Parse Time Error from a dpll CSV sample"""
    id_ = 'dpll/time-error'
    elems = ('timestamp', 'eecstate', 'state', 'terror')
    y_name = 'terror'
    parsed = namedtuple('Parsed', elems)

    def make_parsed(self, elems):
        if len(elems) < len(self.elems):
            raise ValueError(elems)
        timestamp = parse_timestamp(elems[0])
        eecstate = _int_state(elems[1])
        state = _int_state(elems[2])
        terror = parse_decimal(elems[3]) if elems[3] not in (None, '') else Decimal(0)
        return self.parsed(timestamp, eecstate, state, terror)

    def parse_line(self, line):
        # DPLL samples come from a fixed format CSV file
        return self.make_parsed(line.split(','))


class SMA1TimeErrorParser(TimeErrorParser):
    id_ = 'dpll-sma1/time-error'
