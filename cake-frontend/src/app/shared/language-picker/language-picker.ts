import { Component, inject } from '@angular/core';
import { Language, LanguageService } from '../../core/services/language.service';
import { TitleCasePipe } from '@angular/common';

@Component({
  imports: [TitleCasePipe],
  selector: 'app-language-picker',
  styleUrl: './language-picker.css',
  templateUrl: './language-picker.html',
})
export class LanguagePicker {
  private readonly languageService = inject(LanguageService);

  switchLanguage(language: Language) {
    this.languageService.setLanguage(language === 'en' ? 'fr' : 'en');
  }

  getCurrentLanguage(): Language {
    return this.languageService.currentLanguage;
  }
}
