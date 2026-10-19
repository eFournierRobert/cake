import { Component } from '@angular/core';
import { Logout } from '../logout/logout';
import { LanguagePicker } from '../language-picker/language-picker';
import { ProfilePopup } from '../profile-popup/profile-popup';

@Component({
  imports: [Logout, LanguagePicker, ProfilePopup],
  selector: 'app-header',
  styleUrl: './header.css',
  templateUrl: './header.html',
})
export class Header {}
