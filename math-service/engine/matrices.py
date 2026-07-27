from sympy import Matrix

def matrix_operation(operation, matrix_data, extra_data=None):
    if extra_data is None:
        extra_data = {}
    mat = Matrix(matrix_data)
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
        other = Matrix(extra_data.get('matrix2', []))
        result = mat * other
        return {"result": str(result), "latex": str(result)}
    elif operation == 'add':
        other = Matrix(extra_data.get('matrix2', []))
        result = mat + other
        return {"result": str(result), "latex": str(result)}
    else:
        return {"error": f"Unknown operation: {operation}"}
