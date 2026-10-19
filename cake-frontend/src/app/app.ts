import { Component, inject, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { LoginComponent } from './features/login/login.component/login.component';
import { TranslateService } from '@ngx-translate/core';
import { LanguagePicker } from './shared/language-picker/language-picker';

@Component({
  imports: [RouterOutlet, LoginComponent, LanguagePicker],
  selector: 'app-root',
  styleUrl: './app.css',
  templateUrl: './app.html',
})
export class App {
  private translate = inject(TranslateService);

  construction() {
    this.translate.addLangs(['en', 'fr']);
  }
}
