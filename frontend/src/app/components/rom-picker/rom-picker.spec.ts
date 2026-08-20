import { ComponentFixture, TestBed } from '@angular/core/testing';

import { RomPicker } from './rom-picker';

describe('RomPicker', () => {
  let component: RomPicker;
  let fixture: ComponentFixture<RomPicker>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RomPicker]
    })
    .compileComponents();

    fixture = TestBed.createComponent(RomPicker);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
