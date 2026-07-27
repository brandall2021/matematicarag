import pytest
from engine.calculus import differentiate, integrate, compute_limit
from sympy import Symbol, sympify

x = Symbol('x')


def test_differentiate_x_cubed():
    result = differentiate('x**3', 'x')
    assert sympify(result) == 3 * x**2


def test_differentiate_polynomial():
    result = differentiate('x**3 + 2*x**2 - 5*x', 'x')
    assert sympify(result) == 3 * x**2 + 4 * x - 5


def test_integrate_x_squared():
    result = integrate('x**2', 'x')
    assert sympify(result) == x**3 / 3


def test_integrate_linear():
    result = integrate('2*x + 1', 'x')
    assert sympify(result) == x**2 + x


def test_limit_sin_x_over_x():
    from sympy import sin
    result = compute_limit('sin(x)/x', 'x', '0')
    assert sympify(result) == 1


def test_limit_polynomial():
    result = compute_limit('x**2 + 1', 'x', '2')
    assert sympify(result) == 5
