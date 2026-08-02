from sympy import Matrix
from .parser import safe_parse

_NUMERIC_CELL_RE = __import__('re').compile(r'^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$')

def _parse_matrix(data):
    """Build a SymPy Matrix validating every cell. Only plain numbers are
    accepted; arbitrary strings would be sympified by Matrix() using eval."""
    if not isinstance(data, list) or not all(isinstance(row, list) for row in data):
        raise ValueError("matrix must be a 2D array")
    rows = []
    for row in data:
        cells = []
        for cell in row:
            if isinstance(cell, bool) or not isinstance(cell, (int, float, str)):
                raise ValueError("matrix cells must be numbers")
            if isinstance(cell, str):
                if not _NUMERIC_CELL_RE.match(cell.strip()):
                    raise ValueError("matrix cells must be numbers")
                try:
                    cells.append(float(cell))
                except ValueError:
                    raise ValueError("matrix cells must be numbers")
            else:
                cells.append(cell)
        rows.append(cells)
    return Matrix(rows)

def matrix_operation(operation, matrix_data, extra_data=None):
    if extra_data is None:
        extra_data = {}
    mat = _parse_matrix(matrix_data)
    if operation == 'determinant':
        det = mat.det()
        return {"result": str(det), "latex": str(det)}
    elif operation == 'transpose':
        result = mat.T
        return {"result": str(result), "latex": str(result)}
    elif operation == 'inverse':
        result = mat.inv()
        return {"result": str(result), "latex": str(result)}
    elif operation == 'rank':
        result = mat.rank()
        return {"result": str(result), "latex": str(result)}
    elif operation == 'rref':
        result, pivots = mat.rref()
        return {"result": str(result), "latex": str(result), "pivots": list(pivots)}
    elif operation == 'eigenvalues':
        result = mat.eigenvals()
        return {"result": str(result), "latex": str(result)}
    elif operation == 'multiply':
        other = _parse_matrix(extra_data.get('matrix2', []))
        result = mat * other
        return {"result": str(result), "latex": str(result)}
    elif operation == 'add':
        other = _parse_matrix(extra_data.get('matrix2', []))
        result = mat + other
        return {"result": str(result), "latex": str(result)}
    else:
        return {"error": f"Unknown operation: {operation}"}
