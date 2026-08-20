import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GameViewport } from './game-viewport';

describe('GameViewport', () => {
  let component: GameViewport;
  let fixture: ComponentFixture<GameViewport>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [GameViewport]
    })
    .compileComponents();

    fixture = TestBed.createComponent(GameViewport);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
