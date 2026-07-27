import pytest
from engine.verify import verify_result


def test_verify_correct():
    result = verify_result('3*x**2', '3*x**2', 'differentiate')
    assert result['verified'] is True


def test_verify_incorrect():
    result = verify_result('2*x**2', '3*x**2', 'differentiate')
    assert result['verified'] is False
