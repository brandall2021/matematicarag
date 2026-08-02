import { RenderMathPipe } from './render-math.pipe';

describe('RenderMathPipe', () => {
  let pipe: RenderMathPipe;

  beforeEach(() => {
    pipe = new RenderMathPipe();
  });

  it('creates an instance', () => {
    expect(pipe).toBeTruthy();
  });

  it('returns empty string for empty input', () => {
    expect(pipe.transform('')).toBe('');
    expect(pipe.transform(null as unknown as string)).toBe('');
  });

  it('renders inline math delimited by $', () => {
    const html = pipe.transform('Resuelve $x^2$ para $x$');
    expect(html).toContain('katex');
    expect(html).toContain('x^2');
  });

  it('renders display math delimited by $$', () => {
    const html = pipe.transform('$$\\frac{a}{b}$$');
    expect(html).toContain('katex-display');
  });

  it('wraps plain text in a paragraph', () => {
    const html = pipe.transform('hola');
    expect(html).toContain('<p>');
  });

  it('neutralizes script injection attempts', () => {
    const payload = '<script>alert(1)</script>';
    const html = pipe.transform(payload);
    expect(html).not.toContain('<script');
  });

  it('strips event handlers from injected tags', () => {
    const payload = '<img src=x onerror="alert(1)">';
    const html = pipe.transform(payload);
    expect(html).not.toContain('onerror');
  });

  it('neutralizes javascript: URLs', () => {
    const payload = '<a href="javascript:alert(1)">click</a>';
    const html = pipe.transform(payload);
    expect(html).not.toContain('javascript:');
  });
});
