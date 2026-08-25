import { Component, EventEmitter, inject, output } from '@angular/core';
import { HttpService } from '../../services/http.service';
import { environment } from '../../environments/environment';
import { toSignal } from '@angular/core/rxjs-interop';
import { RomDetails } from '../../common/models';

@Component({
  selector: 'app-rom-picker',
  imports: [],
  templateUrl: './rom-picker.html',
  styleUrl: './rom-picker.scss',
})
export class RomPicker {
  httpService_ = inject(HttpService);

  romSelected = output<RomDetails>();

  romLoadSelected: boolean = false;
  fetchedRoms = toSignal(this.httpService_.httpGET<RomDetails[]>(environment.fetchRoms), { initialValue: [] });

  openRoms() {
    this.romLoadSelected = true;
  }

  loadRom(rom: RomDetails) {
    // Implementation for loading the selected ROM
    this.romSelected.emit(rom);
  }
}
