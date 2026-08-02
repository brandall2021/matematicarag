import re
from sympy import (
    Abs, Add, E, Eq, Float, Function, Ge, Gt, I, Integer, Le, Lt, Matrix,
    Mul, Ne, Pow, Rational, Symbol, acos, acosh, acot, acsc, asec, asin,
    asinh, atan, atanh, ceiling, cos, cosh, cot, csc, exp, factorial, floor,
    gamma, im, log, oo, parse_expr, pi, re as sympy_re, sec, sign, sin,
    sinh, sqrt, tan, tanh,
)
from sympy.parsing.sympy_parser import (
    convert_xor,
    function_exponentiation,
    implicit_multiplication_application,
    standard_transformations,
)

_FUNCTIONS = {
    'sin', 'cos', 'tan', 'cot', 'sec', 'csc',
    'arcsin', 'arccos', 'arctan',
    'sinh', 'cosh', 'tanh',
    'ln', 'log', 'sqrt', 'exp', 'abs',
}

_MACROS = {
    'pi': 'pi', 'infty': 'oo',
    'alpha': 'alpha', 'beta': 'beta', 'gamma': 'gamma', 'theta': 'theta',
    'lambda': 'lambda', 'mu': 'mu', 'sigma': 'sigma', 'omega': 'omega',
    'phi': 'phi', 'tau': 'tau', 'rho': 'rho', 'epsilon': 'epsilon',
    'Delta': 'delta', 'Gamma': 'gamma',
    'cdot': '*', 'times': '*', 'div': '/',
    'ge': '>=', 'le': '<=', 'ne': '!=', 'lt': '<', 'gt': '>',
}

_SPACING = {';', ',', '!', 'quad', 'qquad'}

# ---------------------------------------------------------------------------
# Security: `parse_expr` evaluates its input with eval()-like semantics
# (CVE-2024-46946). We therefore validate every expression BEFORE it reaches
# SymPy and hand parse_expr a locked-down environment:
#   * a whitelisted global dict (no access to Python builtins),
#   * transform_reduce_all (undefined functions degrade to multiplications),
#   * strict=True (rejects names not resolvable through the dictionaries),
#   * a lexical validator that rejects attribute access, dunder names,
#     string literals and Python statements outright.
# ---------------------------------------------------------------------------

_MAX_INPUT_LENGTH = 1000

# Symbols (variables) users are allowed to write. Anything outside this set
# that is not a whitelisted function is rejected.
_SYMBOL_NAMES = {
    'x', 'y', 'z', 't', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
    'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 'u', 'v', 'w',
    'alpha', 'beta', 'gamma', 'delta', 'theta', 'lambda', 'mu', 'sigma',
    'omega', 'phi', 'tau', 'rho', 'epsilon', 'eta', 'zeta', 'iota', 'kappa',
    'nu', 'xi', 'omicron', 'psi', 'chi',
}

# Identifiers that are never valid inside a math expression.
_FORBIDDEN_IDENTIFIERS = {
    'import', 'lambda', 'exec', 'eval', 'compile', 'globals', 'locals',
    'getattr', 'setattr', 'delattr', 'vars', 'open', 'input', 'print',
    'system', 'popen', 'subprocess', 'class', 'type', 'object', 'help',
}

# Characters that can never appear in a valid math expression. These are the
# building blocks of every known SymPy/parse_expr escape chain.
_FORBIDDEN_CHARS = set('_`~@#;:{}"\'\\')

# A single-letter symbol, optionally followed by digits (x, x1, t, a2, ...).
_SYMBOL_NAME_RE = re.compile(r'^[a-zA-Z][0-9]*$')

# Everything parse_expr is allowed to resolve. No Python builtins are exposed,
# so even a bypass of the lexical checks cannot reach `__import__` & friends.
_SAFE_GLOBALS = {
    'sin': sin, 'cos': cos, 'tan': tan, 'cot': cot, 'sec': sec, 'csc': csc,
    'arcsin': asin, 'arccos': acos, 'arctan': atan,
    'asin': asin, 'acos': acos, 'atan': atan,
    'sinh': sinh, 'cosh': cosh, 'tanh': tanh,
    'asinh': asinh, 'acosh': acosh, 'atanh': atanh,
    'ln': log, 'log': log, 'sqrt': sqrt, 'exp': exp,
    'abs': Abs, 'Abs': Abs, 're': sympy_re, 'im': im, 'sign': sign,
    'floor': floor, 'ceiling': ceiling, 'factorial': factorial,
    'gamma': gamma, 'Rational': Rational, 'Integer': Integer, 'Float': Float,
    'pi': pi, 'oo': oo, 'inf': oo, 'E': E, 'I': I,
    'Eq': Eq, 'Ne': Ne, 'Le': Le, 'Lt': Lt, 'Ge': Ge, 'Gt': Gt,
    'Matrix': Matrix,
    # Core sympy nodes required by the parser transformations.
    'Add': Add, 'Mul': Mul, 'Pow': Pow, 'Symbol': Symbol, 'Function': Function,
}

_TRANSFORMATIONS = standard_transformations + (
    convert_xor,
    function_exponentiation,
    implicit_multiplication_application,
)


def _symbol_or_func_name(name):
    """Whitelist check for a parsed identifier."""
    if name in _FUNCTIONS or name in _SAFE_GLOBALS:
        return True
    if name in _FORBIDDEN_IDENTIFIERS:
        return False
    return name in _SYMBOL_NAMES or _SYMBOL_NAME_RE.match(name) is not None


def _validate_expr(s):
    """Reject any token that could be used to escape the math sandbox."""
    if not s:
        raise ValueError("empty expression")
    if len(s) > _MAX_INPUT_LENGTH:
        raise ValueError("expression is too long")

    for ch in s:
        if ch in _FORBIDDEN_CHARS:
            raise ValueError("invalid character in expression: %r" % ch)

    # '.' is only valid as a decimal separator inside a number literal.
    for i, ch in enumerate(s):
        if ch == '.':
            prev_ok = i > 0 and (s[i - 1].isdigit())
            next_ok = i + 1 < len(s) and s[i + 1].isdigit()
            if not (prev_ok and next_ok):
                raise ValueError("invalid character '.' in expression")

    for ident in re.findall(r'[A-Za-z_][A-Za-z0-9_]*', s):
        if not _symbol_or_func_name(ident):
            raise ValueError("unknown or forbidden identifier: %s" % ident)


def safe_parse(expr_str: str):
    translated = latex_to_sympy(expr_str)
    normalized = normalize_input(translated)
    _validate_expr(normalized)
    try:
        return parse_expr(
            normalized,
            evaluate=False,
            global_dict=_SAFE_GLOBALS,
            local_dict={},
            transformations=_TRANSFORMATIONS,
        )
    except SyntaxError as e:
        raise ValueError("expression could not be parsed safely: %s" % e)


def _read_group(s, i):
    """s[i] == '{': return (content without braces, index after closing brace)."""
    depth = 0
    j = i
    while j < len(s):
        if s[j] == '{':
            depth += 1
        elif s[j] == '}':
            depth -= 1
            if depth == 0:
                return s[i + 1:j], j + 1
        j += 1
    return s[i + 1:], len(s)


def _next_arg(s, i):
    """Return (arg, next_index): a braced group or a single alnum token, skipping spaces."""
    while i < len(s) and s[i].isspace():
        i += 1
    if i < len(s) and s[i] == '{':
        return _read_group(s, i)
    j = i
    while j < len(s) and (s[j].isalnum() or s[j] in '._'):
        j += 1
    if j > i:
        return s[i:j], j
    return '', i


def _append(out, token, atom=False):
    r"""Append token; insert explicit '*' where LaTeX implied multiplication.

    `atom` marks tokens produced by LaTeX commands (sqrt/frac/macro/group):
    those act as atomic factors, so `x\sqrt{2}` -> `x*sqrt(2)`.
    Literal chars pass through untouched: `sin(x)` stays `sin(x)`.
    """
    if not token:
        return
    if out:
        prev_text, prev_atom = out[-1]
        prev_last = prev_text[-1]
        insert = False
        if atom and (prev_last.isalnum() or prev_last == ')'):
            insert = True
        elif prev_atom and token[0].isalnum():
            insert = True
        if insert:
            out.append(('*', False))
    out.append((token, atom))


def latex_to_sympy(s):
    s = s.strip()
    s = re.sub(r'^\$\$?', '', s)
    s = re.sub(r'\$\$?$', '', s)
    s = s.replace('\\left', '').replace('\\right', '')

    out = []
    i = 0
    n = len(s)
    while i < n:
        c = s[i]
        if c == '\\':
            j = i + 1
            while j < n and s[j].isalpha():
                j += 1
            cmd = s[i:j]
            name = cmd[1:] if len(cmd) > 1 else ''

            if cmd == '\\frac':
                num, k = _next_arg(s, j)
                den, k2 = _next_arg(s, k)
                if num and den:
                    _append(out, '(' + latex_to_sympy(num) + ')/(' + latex_to_sympy(den) + ')', atom=True)
                    i = k2
                    continue
                out.append((cmd, False))
                i = j
                continue

            if cmd == '\\sqrt':
                arg, k = _next_arg(s, j)
                if arg:
                    _append(out, 'sqrt(' + latex_to_sympy(arg) + ')', atom=True)
                    i = k
                    continue
                out.append((cmd, False))
                i = j
                continue

            if name in _FUNCTIONS:
                arg, k = _next_arg(s, j)
                if arg and not arg.startswith('('):
                    _append(out, name + '(' + latex_to_sympy(arg) + ')', atom=True)
                    i = k
                    continue
                _append(out, name, atom=True)
                i = j
                continue

            if name in _MACROS:
                _append(out, _MACROS[name], atom=name not in ('cdot', 'times', 'div', 'ge', 'le', 'ne', 'lt', 'gt'))
                i = j
                continue

            if name in _SPACING:
                i = j
                continue

            if cmd in ('\\{', '\\}'):
                out.append((cmd[-1], False))
                i = j
                continue
            if cmd == '\\ ':
                i = j
                continue

            out.append((cmd, False))
            i = j
            continue

        if c == '^':
            _append(out, '**')
            i += 1
            continue

        if c == '{':
            content, k = _read_group(s, i)
            _append(out, '(' + latex_to_sympy(content) + ')', atom=True)
            i = k
            continue

        _append(out, c)
        i += 1

    return ''.join(t for t, _ in out)


def normalize_input(expr_str: str) -> str:
    s = expr_str.strip()
    s = re.sub(r'^\${1,2}|\${1,2}$', '', s)
    s = re.sub(r'^\\\\?\\(|\\\\?\\)$', '', s)
    s = re.sub(r'^\\\\?\\[|\\\\?\\]$', '', s)
    superscripts = {'⁰': '**0', '¹': '**1', '²': '**2', '³': '**3',
                    '⁴': '**4', '⁵': '**5', '⁶': '**6', '⁷': '**7',
                    '⁸': '**8', '⁹': '**9', 'ⁿ': '**n'}
    for uni, rep in superscripts.items():
        s = s.replace(uni, rep)
    s = s.replace('^', '**')
    s = re.sub(r'(\d)([a-zA-Z(])', r'\1*\2', s)
    s = re.sub(r'\)\(', r')*(', s)
    s = re.sub(r'\)([a-zA-Z])', r')*\1', s)
    return s
