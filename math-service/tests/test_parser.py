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


@pytest.mark.parametrize('latex,expected', [
    (r'\int_0^{\infty}\!3x\,\mathrm{d}x', 'Integral(3*x, (x, 0, oo))'),
    (r'\int_0^1 x^2\, dx', 'Integral(x**2, (x, 0, 1))'),
    (r'\int x^2 dx', 'Integral(x**2, x)'),
    (r'\int_0^1 x \, \mathrm{d}x', 'Integral(x, (x, 0, 1))'),
    (r'\int_0^\infty e^{-x} dx', 'Integral(e**(-x), (x, 0, oo))'),
    (r'\int_1^2 \frac{1}{x} dx', 'Integral((1)/(x), (x, 1, 2))'),
])
def test_integral_latex_to_sympy(latex, expected):
    assert normalize_input(latex_to_sympy(latex)) == expected


@pytest.mark.parametrize('latex,expected', [
    (r'\int_0^{\infty}\!3x\,\mathrm{d}x', 'oo'),
    (r'\int_0^1 x^2\, dx', '1/3'),
    (r'\int x^2 dx', 'x**3/3'),
    (r'\int_0^1 x \, \mathrm{d}x', '1/2'),
    (r'\int_0^\infty e^{-x} dx', '1'),
    (r'\int_1^2 \frac{1}{x} dx', 'log(2)'),
    (r'\int x \sin(x) dx', '-x*cos(x) + sin(x)'),
    (r'\int_0^1 x dx + \int_1^2 x dx', '2'),
])
def test_integral_safe_parse(latex, expected):
    result = safe_parse(latex).doit()
    assert result == sympify(expected)


def test_e_is_eulers_number():
    assert safe_parse(r'e^{x}').doit() == sympify('exp(x)')
    assert safe_parse('2*e').doit() == sympify('2*E')


def test_spacing_commands_do_not_leak_backslashes():
    assert latex_to_sympy(r'2\,x') == '2x'
    assert latex_to_sympy(r'a\!b') == 'ab'
    assert normalize_input(latex_to_sympy(r'2\,x')) == '2*x'
    assert '\\' not in latex_to_sympy(r'\int_0^1 x\,dx')
