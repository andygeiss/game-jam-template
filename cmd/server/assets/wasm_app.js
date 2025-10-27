const go = new Go(); // Defined in wasm_exec.js
const WASM_URL = "/assets/game.wasm";
var wasm;

WebAssembly.instantiateStreaming(fetch(WASM_URL), go.importObject).then(
  function (obj) {
    wasm = obj.instance;
    go.run(wasm);
  },
);
