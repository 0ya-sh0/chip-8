const serverUrl = 'http://localhost:9000';
const wsServerUrl = 'ws://localhost:9000';

export const environment = {
  production: false,


  fetchRoms: serverUrl + '/game',
  connectToGame: wsServerUrl + '/game/{id}',
};