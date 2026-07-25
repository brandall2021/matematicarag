import { Injectable, signal, effect } from '@angular/core';

export type Theme = 'dark' | 'light';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  currentTheme = signal<Theme>((localStorage.getItem('theme') as Theme) || 'dark');

  constructor() {
    effect(() => {
      const theme = this.currentTheme();
      document.documentElement.setAttribute('data-theme', theme);
      localStorage.setItem('theme', theme);
    });
  }

  toggle(): void {
    this.currentTheme.set(this.currentTheme() === 'dark' ? 'light' : 'dark');
  }
}
