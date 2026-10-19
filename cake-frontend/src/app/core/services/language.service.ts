import { inject, Service } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';

export type Language = 'en' | 'fr';

@Service()
export class LanguageService {
  private readonly translate = inject(TranslateService);

  setLanguage(lang: Language) {
    localStorage.setItem('language', lang);
    this.translate.use(lang);
  }

  get currentLanguage(): Language {
    return this.translate.getCurrentLang() as Language;
  }
}
