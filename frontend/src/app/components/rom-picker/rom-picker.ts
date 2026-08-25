import { Component, inject, output } from '@angular/core';
import { environment } from '../../environments/environment';
import { toSignal } from '@angular/core/rxjs-interop';
import { RomDetails } from '../../common/models';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-rom-picker',
  imports: [],
  templateUrl: './rom-picker.html',
  styleUrl: './rom-picker.scss',
})
export class RomPicker {
  httpClient_ = inject(HttpClient);

  romSelected = output<RomDetails>();

  romLoadSelected: boolean = false;
  fetchedRoms = toSignal(this.httpClient_.get<RomDetails[]>(environment.fetchRoms), { initialValue: [] });

  openRoms() {
    this.romLoadSelected = true;
  }

  loadRom(rom: RomDetails) {
    // Implementation for loading the selected ROM
    this.romSelected.emit(rom);
  }
}
