import { ComponentFixture, TestBed } from '@angular/core/testing';
import { LanguagePicker } from './language-picker';

describe('LanguagePicker', () => {
  let component: LanguagePicker;
  let fixture: ComponentFixture<LanguagePicker>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LanguagePicker],
    }).compileComponents();

    fixture = TestBed.createComponent(LanguagePicker);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
