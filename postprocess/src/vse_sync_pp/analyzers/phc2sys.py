### SPDX-License-Identifier: GPL-2.0-or-later

"""Analyze phc2sys log messages"""

from .analyzer import TimeErrorAnalyzerBase
from .analyzer import TimeDeviationAnalyzerBase
from .analyzer import MaxTimeIntervalErrorAnalyzerBase


def _round_ns(val):
    """Return offset/limit values suitable for report tables."""
    try:
        rounded = round(val.item(), 3)
    except AttributeError:
        rounded = round(val, 3)
    if rounded == int(rounded):
        return int(rounded)
    return rounded


def phc2sys_time_error_diagnostics(data, limit_ns, locked_states, transient_period_s):
    """Return PHC-to-SYS triage fields for the test report analysis section."""
    if len(data) == 0:
        return {}

    terror = data.terror
    abs_terror = terror.abs()
    max_abs = abs_terror.max()
    worst_idx = abs_terror.idxmax()
    locked_mask = data.state.isin(locked_states)

    return {
        'sample_count': int(len(data)),
        'limit_ns': _round_ns(limit_ns),
        'max_abs_offset_ns': _round_ns(max_abs),
        'worst_offset_ns': _round_ns(terror.loc[worst_idx]),
        'samples_exceeding_limit': int((abs_terror >= limit_ns).sum()),
        'samples_not_locked': int((~locked_mask).sum()),
        'states_observed': ','.join(str(state) for state in sorted(data.state.unique(), key=str)),
        'transient_period_s': transient_period_s,
    }


class TimeErrorAnalyzer(TimeErrorAnalyzerBase):
    """Analyze time error"""
    id_ = 'phc2sys/time-error'
    parser = id_
    locked = frozenset({'s2'})

    def test(self, data):
        return self._check_missing_samples(data, *super().test(data))

    def explain(self, data):
        analysis = super().explain(data)
        if not analysis:
            return analysis
        analysis['diagnostics'] = phc2sys_time_error_diagnostics(
            data,
            self._unacceptable,
            self.locked,
            self._transient,
        )
        return analysis


class TimeDeviationAnalyzer(TimeDeviationAnalyzerBase):
    """Analyze time deviation"""
    id_ = 'phc2sys/time-deviation'
    parser = 'phc2sys/time-error'
    locked = frozenset({'s2'})

    def test(self, data):
        return self._check_missing_samples(data, *super().test(data))


class MaxTimeIntervalErrorAnalyzer(MaxTimeIntervalErrorAnalyzerBase):
    """Analyze max time interval error"""
    id_ = 'phc2sys/mtie'
    parser = 'phc2sys/time-error'
    locked = frozenset({'s2'})

    def test(self, data):
        return self._check_missing_samples(data, *super().test(data))
