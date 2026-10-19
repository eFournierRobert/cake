import { Component, inject, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { LoginComponent } from './features/login/login.component/login.component';
import { TranslateService } from '@ngx-translate/core';
import { LanguagePicker } from './shared/language-picker/language-picker';
import { Logout } from './shared/logout/logout';

@Component({
  imports: [RouterOutlet],
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
