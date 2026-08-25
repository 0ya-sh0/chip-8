import { Component, EventEmitter, inject, output } from '@angular/core';
import { HttpService } from '../../services/http.service';
import { environment } from '../../environments/environment';
import { toSignal } from '@angular/core/rxjs-interop';

@Component({
  selector: 'app-rom-picker',
  imports: [],
  templateUrl: './rom-picker.html',
  styleUrl: './rom-picker.scss',
})
export class RomPicker {
  httpService_ = inject(HttpService);

  romSelected = output<any>();

  romLoadSelected: boolean = false;
  fetchedRoms: any = toSignal(this.httpService_.httpGET(environment.fetchRoms), { initialValue: null });

  openRoms() {
    this.romLoadSelected = true;
  }

  loadRom(rom: any) {
    // Implementation for loading the selected ROM
    this.romSelected.emit(rom);
  }
}
