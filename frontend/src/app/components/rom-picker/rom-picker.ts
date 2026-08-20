import { ChangeDetectorRef, Component, EventEmitter, inject, Output } from '@angular/core';
import { NgIf } from "../../../../node_modules/@angular/common/types/_common_module-chunk";
import { firstValueFrom } from 'rxjs/internal/firstValueFrom';
import { HttpService } from '../../services/http.service';
import { environment } from '../../environments/environment';

@Component({
  selector: 'app-rom-picker',
  imports: [],
  templateUrl: './rom-picker.html',
  styleUrl: './rom-picker.scss',
})
export class RomPicker {
  cdr = inject(ChangeDetectorRef);
  romLoadSelected: boolean = false;
  fetchedRoms: any = [];
  httpService_ = inject(HttpService);

  @Output() romSelected = new EventEmitter<any>();

  ngOnInit() {
    this.fetchRoms();
  }

  openRoms() {
    this.romLoadSelected = true;
  }

  async fetchRoms() {
    // Fetch the list of ROMs from the backend
    this.fetchedRoms = await firstValueFrom(this.httpService_.httpGET(environment.fetchRoms));
    this.cdr.detectChanges();
  }

  loadRom(rom: any) {
    // Implementation for loading the selected ROM
    this.romSelected.emit(rom);
  }
}
