import pytest
from engine.parser import latex_to_sympy, safe_parse, normalize_input
from sympy import sympify


@pytest.mark.parametrize('latex,expected', [
    ('\\sqrt{3}', 'sqrt(3)'),
    ('\\sqrt3', 'sqrt(3)'),
    ('\\frac{1}{2}', '(1)/(2)'),
    ('\\frac{x+1}{x-1}', '(x+1)/(x-1)'),
    ('x^{2}', 'x**(2)'),
    ('2\\cdot3', '2*3'),
    ('2\\times3', '2*3'),
    ('\\pi', 'pi'),
    ('\\left(x\\right)', '(x)'),
    ('\\sin(x)', 'sin(x)'),
    ('\\sin x', 'sin(x)'),
    ('\\sqrt{\\frac{1}{4}}', 'sqrt((1)/(4))'),
    ('x\\sqrt{2}', 'x*sqrt(2)'),
    ('\\frac{3}{2}x', '(3)/(2)*x'),
    ('x^{2}+1', 'x**(2)+1'),
])
def test_latex_to_sympy(latex, expected):
    assert latex_to_sympy(latex) == expected


@pytest.mark.parametrize('latex,expected', [
    ('\\sqrt{9}', '3'),
    ('\\sqrt3', 'sqrt(3)'),
    ('\\frac{1}{2}', '1/2'),
    ('2+2', '4'),
    ('x\\sqrt{2}', 'sqrt(2)*x'),
    ('\\sqrt{\\frac{1}{4}}', '1/2'),
    ('\\frac{x+1}{x-1}', '(x+1)/(x-1)'),
])
def test_safe_parse_handles_latex(latex, expected):
    result = safe_parse(latex).doit()
    assert result == sympify(expected)


def test_plain_sympy_input_still_works():
    assert safe_parse('x**3').doit() == sympify('x**3')
    assert safe_parse('sin(x)/x').doit() == sympify('sin(x)/x')
    assert safe_parse('x^2 + 5*x + 6').doit() == sympify('x**2 + 5*x + 6')


def test_normalize_preserves_function_calls():
    assert normalize_input('sin(x)') == 'sin(x)'
    assert normalize_input('(x+1)(x+2)') == '(x+1)*(x+2)'
