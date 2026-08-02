import { SafeHtmlPipe } from './safe-html.pipe';

describe('SafeHtmlPipe', () => {
  let pipe: SafeHtmlPipe;

  beforeEach(() => {
    pipe = new SafeHtmlPipe();
  });

  it('creates an instance', () => {
    expect(pipe).toBeTruthy();
  });

  it('returns empty string for empty input', () => {
    expect(pipe.transform('')).toBe('');
    expect(pipe.transform(null as unknown as string)).toBe('');
  });

  it('keeps safe HTML', () => {
    const html = pipe.transform('<p>2 + 2 = 4</p>');
    expect(html).toContain('<p>2 + 2 = 4</p>');
  });

  it('removes script tags', () => {
    expect(pipe.transform('<script>alert(1)</script>')).not.toContain('<script');
  });

  it('strips event handlers', () => {
    expect(pipe.transform('<img src=x onerror="alert(1)">')).not.toContain('onerror');
  });

  it('strips javascript: URLs', () => {
    expect(pipe.transform('<a href="javascript:alert(1)">x</a>')).not.toContain('javascript:');
  });
});
