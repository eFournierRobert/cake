import { Component, inject } from '@angular/core';
import { AuthService, LoginBody } from '../../../core/services/auth.service';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { submit } from '@angular/forms/signals';
import { TranslatePipe } from '@ngx-translate/core';
import { Router } from '@angular/router';

@Component({
  imports: [ReactiveFormsModule, TranslatePipe],
  selector: 'app-login',
  styleUrl: './login.component.css',
  templateUrl: './login.component.html',
})
export class LoginComponent {
  private authService: AuthService = inject(AuthService);
  private router = inject(Router);

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
      next: () => {
        this.authService.setAuthenticated(true);
        this.router.navigate(['spaces']);
      },
      error: error => console.log(error),
    });
  }

  protected readonly submit = submit;
}
