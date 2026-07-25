import { Pipe, PipeTransform } from '@angular/core';
import katex from 'katex';

@Pipe({ name: 'renderMath', standalone: true })
export class RenderMathPipe implements PipeTransform {
  transform(text: string): string {
    if (!text) return '';

    // Render display math first: $$...$$
    let result = text.replace(/\$\$([\s\S]+?)\$\$/g, (_, tex) => {
      try {
        return katex.renderToString(tex.trim(), { displayMode: true, throwOnError: false });
      } catch { return tex; }
    });

    // Render inline math: $...$
    result = result.replace(/\$([^\$\n]+?)\$/g, (_, tex) => {
      try {
        return katex.renderToString(tex.trim(), { displayMode: false, throwOnError: false });
      } catch { return tex; }
    });

    // Render \(...\) inline
    result = result.replace(/\\\((.+?)\\\)/g, (_, tex) => {
      try {
        return katex.renderToString(tex.trim(), { displayMode: false, throwOnError: false });
      } catch { return tex; }
    });

    // Render \[...\] display
    result = result.replace(/\\\[(.+?)\\\]/gs, (_, tex) => {
      try {
        return katex.renderToString(tex.trim(), { displayMode: true, throwOnError: false });
      } catch { return tex; }
    });

    // Convert newlines to <br> and wrap in paragraphs
    result = result.replace(/\n\n+/g, '</p><p>');
    result = result.replace(/\n/g, '<br>');

    return `<p>${result}</p>`;
  }
}
