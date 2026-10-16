import { Component, inject } from '@angular/core';
import { AuthService, LoginBody } from '../../../core/services/auth.service';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { submit } from '@angular/forms/signals';

@Component({
  imports: [ReactiveFormsModule],
  selector: 'app-login',
  styleUrl: './login.component.css',
  templateUrl: './login.component.html',
})
export class LoginComponent {
  private authService: AuthService = inject(AuthService);

  loginForm = new FormGroup({
    username: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
    password: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
  });

  onSubmit() {
    if (this.loginForm.invalid) {
      return;
    }

    const body = this.loginForm.value;

    this.authService.login(<LoginBody>body).subscribe({
      next: () => console.log('logged in!'),
      error: error => console.log(error),
    });
  }

  protected readonly submit = submit;
}
