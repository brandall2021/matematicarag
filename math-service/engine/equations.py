from sympy import Eq, solve, Symbol
from engine.parser import normalize_input, safe_parse

def solve_equation(expr_str, var_str='x'):
    var = Symbol(var_str)
    normalized = normalize_input(expr_str)
    if '=' in normalized:
        parts = normalized.split('=', 1)
        lhs = safe_parse(parts[0])
        rhs = safe_parse(parts[1])
        equation = Eq(lhs, rhs)
    else:
        equation = safe_parse(normalized)
    solutions = solve(equation, var)
    solution_strs = [str(s) for s in solutions]
    return {
        "variables": [var_str],
        "solutions": solution_strs,
        "latex": " , ".join(solution_strs),
        "count": len(solutions),
    }

def solve_system(equations_str, var_str='x'):
    vars_list = [Symbol(v.strip()) for v in var_str.split(',')]
    eqs = []
    for eq_str in equations_str:
        normalized = normalize_input(eq_str)
        if '=' in normalized:
            parts = normalized.split('=', 1)
            lhs = safe_parse(parts[0])
            rhs = safe_parse(parts[1])
            eqs.append(Eq(lhs, rhs))
        else:
            eqs.append(safe_parse(normalized))
    solution = solve(eqs, vars_list, dict=True)
    result_vars = {}
    if solution:
        for v in vars_list:
            result_vars[str(v)] = str(solution[0].get(v, 'unknown'))
    return {
        "variables": [str(v) for v in vars_list],
        "solutions": result_vars,
        "latex": ", ".join([f"{k} = {v}" for k, v in result_vars.items()]),
        "method": "symbolic",
    }
