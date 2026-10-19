import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ProfilePopup } from './profile-popup';

describe('ProfilePopup', () => {
  let component: ProfilePopup;
  let fixture: ComponentFixture<ProfilePopup>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfilePopup],
    }).compileComponents();

    fixture = TestBed.createComponent(ProfilePopup);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
