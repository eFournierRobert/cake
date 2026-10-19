import { Component, inject, signal } from '@angular/core';
import { User, UserService } from '../../core/services/user.service';
import { LanguagePicker } from '../language-picker/language-picker';
import { Logout } from '../logout/logout';

@Component({
  imports: [LanguagePicker, Logout],
  selector: 'app-profile-popup',
  styleUrl: './profile-popup.css',
  templateUrl: './profile-popup.html',
})
export class ProfilePopup {
  private readonly userService = inject(UserService);
  protected readonly currentUser = signal<User | undefined>(undefined);
  protected isOpen: boolean = false;

  constructor() {
    this.userService.getCurrentUser().subscribe((currentUser) => {
      this.currentUser.set(currentUser as User);
    });
  }

  toggleMenu() {
    this.isOpen = !this.isOpen;
  }
}
