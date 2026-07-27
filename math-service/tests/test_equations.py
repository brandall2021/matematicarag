import pytest
from engine.equations import solve_equation


def test_quadratic():
    result = solve_equation('x^2 + 5*x + 6 = 0', 'x')
    solutions = [complex(s) for s in result['solutions']]
    assert -2 in solutions
    assert -3 in solutions


def test_linear():
    result = solve_equation('2*x + 4 = 0', 'x')
    assert '-2' in result['solutions']
