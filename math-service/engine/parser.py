import re
from sympy import parse_expr
from sympy.parsing.latex import parse_latex

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
    return s

def safe_parse(expr_str: str):
    normalized = normalize_input(expr_str)
    if '\\' in expr_str or '{' in expr_str:
        try:
            return parse_latex(expr_str)
        except Exception:
            pass
    return parse_expr(normalized, evaluate=False)
