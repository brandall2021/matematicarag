import pytest
from engine.plot import plot_expression


def test_linear_plot_points():
    pts = plot_expression('2*x+1', -1, 1, 5)['points']
    assert len(pts) == 5
    assert pts[0] == [-1.0, -1.0]
    assert pts[-1] == [1.0, 3.0]


def test_plot_accepts_latex_input():
    pts = plot_expression(r'x^{2}', -2, 2, 5)['points']
    assert [p[1] for p in pts] == [4.0, 1.0, 0.0, 1.0, 4.0]


def test_plot_inserts_gap_at_singularity():
    pts = plot_expression('1/x', -1, 1, 3)['points']
    assert pts[0] == [-1.0, -1.0]
    assert pts[1][1] is None
    assert pts[2] == [1.0, 1.0]


def test_plot_defaults_variable_to_x():
    result = plot_expression('sin(x)', -1, 1, 3)
    assert result['variable'] == 'x'
    assert len(result['points']) == 3


def test_plot_uses_first_symbol_when_no_x():
    result = plot_expression('t**2', 0, 2, 3)
    assert result['variable'] == 't'
    assert [p[1] for p in result['points']] == [0.0, 1.0, 4.0]


def test_plot_invalid_expression_raises():
    with pytest.raises(ValueError):
        plot_expression('2+', -1, 1)


def test_plot_rejects_inverted_range():
    with pytest.raises(ValueError):
        plot_expression('x', 5, 1)


def test_plot_clamps_point_count():
    result = plot_expression('x', 0, 1, 99999)
    assert len(result['points']) == 2000
