import { Pipe, PipeTransform } from '@angular/core';
import DOMPurify from 'dompurify';

@Pipe({ name: 'safeHtml', standalone: true })
export class SafeHtmlPipe implements PipeTransform {
  transform(html: string): string {
    if (!html) return '';
    return DOMPurify.sanitize(html);
  }
}
