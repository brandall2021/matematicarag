import re
from sympy import parse_expr

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


def safe_parse(expr_str: str):
    translated = latex_to_sympy(expr_str)
    normalized = normalize_input(translated)
    return parse_expr(normalized, evaluate=False)
