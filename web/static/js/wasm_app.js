// Loads the game. The version comes from <html data-version>, so the WASM
// file is fetched fresh after every deploy despite the immutable cache.
const go = new Go(); // defined in wasm_exec.js
const version = document.documentElement.dataset.version || "";
const url = "/static/game.wasm" + (version ? "?v=" + version : "");

WebAssembly.instantiateStreaming(fetch(url), go.importObject).then((result) => {
  go.run(result.instance);
});
