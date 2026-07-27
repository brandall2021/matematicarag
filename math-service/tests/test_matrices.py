import pytest
from engine.matrices import matrix_operation


def test_determinant():
    result = matrix_operation('determinant', [[1, 2], [3, 4]])
    assert result['result'] == '-2'


def test_transpose():
    result = matrix_operation('transpose', [[1, 2], [3, 4]])
    assert '1' in result['result'] and '3' in result['result']


def test_rank():
    result = matrix_operation('rank', [[1, 0], [0, 1]])
    assert result['result'] == '2'
