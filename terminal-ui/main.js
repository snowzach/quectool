import { Terminal } from '@xterm/xterm';
import { AttachAddon } from '@xterm/addon-attach';
import { FitAddon } from '@xterm/addon-fit';

const terminal = new Terminal({convertEol: true});
terminal.open(document.getElementById('terminal'));

const fitAddon = new FitAddon();
terminal.loadAddon(fitAddon);
fitAddon.fit();

const socket = new WebSocket(`/api/terminal?LINES=${terminal.rows}&COLUMNS=${terminal.cols}&TERM=vt100`);
const attachAddon = new AttachAddon(socket);
terminal.loadAddon(attachAddon);
