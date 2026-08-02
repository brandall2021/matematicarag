import pytest
from engine.parser import safe_parse


# Known parse_expr / sympify escape payloads. None of them may be accepted.
RCE_PAYLOADS = [
    "__import__('os').system('id')",
    "().__class__.__bases__[0].__subclasses__()",
    "getattr(__builtins__, '__import__')('os').system('id')",
    "open('/etc/passwd').read()",
    "exec('print(1)')",
    "eval('1+1')",
    "compile('1', '<s>', 'exec')",
    "(lambda: __import__('os').system('id'))()",
    "[x for x in ()]",
    "x.__class__.__mro__[1].__subclasses__()",
    "print(1)",
    "1; import os",
    "max.__self__.__class__.__mro__",
]


@pytest.mark.parametrize("payload", RCE_PAYLOADS)
def test_rce_payloads_rejected(payload):
    with pytest.raises((ValueError, SyntaxError)):
        safe_parse(payload)


def test_unknown_symbol_rejected():
    with pytest.raises(ValueError):
        safe_parse("foobarbaz + 1")


def test_attribute_access_rejected():
    with pytest.raises(ValueError):
        safe_parse("x.attr")


def test_string_literal_rejected():
    with pytest.raises(ValueError):
        safe_parse("x + 'y'")


def test_underscore_rejected():
    with pytest.raises(ValueError):
        safe_parse("__builtins__")


def test_oversized_expression_rejected():
    with pytest.raises(ValueError):
        safe_parse("1 + " * 2000 + "1")


def test_legit_expressions_still_parse():
    assert safe_parse("x**2 + 5*x + 6") is not None
    assert safe_parse("sin(x)/x") is not None
    assert safe_parse("3.14 * x") is not None
    assert safe_parse("\\sqrt{9}") is not None
    assert safe_parse("\\frac{1}{2}") is not None
    assert safe_parse("x\\sqrt{2}") is not None
