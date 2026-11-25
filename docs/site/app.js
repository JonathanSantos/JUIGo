// Site do JUIGo: monta a seção de componentes a partir de cenas.js (gerado
// pela lib), destaca o código Go com as cores do CodeEditor, alterna o tema
// trocando cada captura pela renderização escura e marca a seção visível.
(function () {
  "use strict";

  var raiz = document.documentElement;

  // ---- Destaque de sintaxe (as mesmas classes de token do juigo/syntax) ----
  var PALAVRAS = new Set(("break case chan const continue default defer else fallthrough for func go goto if " +
    "import interface map package range return select struct switch type var").split(" "));
  var EMBUTIDOS = new Set(("append cap close copy delete len make new panic print println recover " +
    "true false nil iota bool byte error float32 float64 int int8 int16 int32 int64 rune string " +
    "uint uint8 uint16 uint32 uint64 any").split(" "));
  var TOKEN = /(\/\/[^\n]*)|("(?:[^"\\\n]|\\.)*"|`[^`]*`|'(?:[^'\\\n]|\\.)*')|(\b\d[\d_]*(?:\.\d+)?\b)|([A-Za-z_]\w*)/g;

  function escapa(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }

  function destacaGo(codigo) {
    var saida = "", ultimo = 0, m;
    TOKEN.lastIndex = 0;
    while ((m = TOKEN.exec(codigo))) {
      saida += escapa(codigo.slice(ultimo, m.index));
      var classe = m[1] ? "c" : m[2] ? "s" : m[3] ? "n" :
        PALAVRAS.has(m[4]) ? "k" : EMBUTIDOS.has(m[4]) ? "b" : "";
      saida += classe ? '<span class="tk-' + classe + '">' + escapa(m[0]) + "</span>" : escapa(m[0]);
      ultimo = TOKEN.lastIndex;
    }
    return saida + escapa(codigo.slice(ultimo));
  }

  // ---- Componentes ----
  function montaComponentes() {
    var cenas = window.CENAS || [];
    var lista = document.getElementById("lista-componentes");
    var indice = document.getElementById("indice-componentes");
    var grupos = [];
    cenas.forEach(function (c) {
      var g = grupos.find(function (x) { return x.nome === c.grupo; });
      if (!g) grupos.push(g = { nome: c.grupo, cenas: [] });
      g.cenas.push(c);
    });

    var htmlIndice = '<p class="indice-grupo">Componentes</p>';
    var html = "";
    grupos.forEach(function (g) {
      html += '<div class="grupo-componentes"><h3>' + escapa(g.nome) + "</h3>";
      g.cenas.forEach(function (c) {
        var id = "c-" + c.id;
        htmlIndice += '<a href="#' + id + '">' + escapa(c.titulo) + "</a>";
        html +=
          '<article class="componente" id="' + id + '">' +
            "<div>" +
              "<h4>" + escapa(c.titulo) + "</h4>" +
              "<p>" + escapa(c.descricao) + "</p>" +
            "</div>" +
            '<figure class="componente-captura">' +
              '<img class="captura" loading="lazy" width="' + c.w + '" height="' + c.h + '"' +
              ' alt="' + escapa(c.titulo) + ' renderizado pela JUIGo"' +
              ' data-claro="componentes/' + c.id + '.png" data-escuro="componentes/' + c.id + '-escuro.png"' +
              ' src="componentes/' + c.id + '.png">' +
            "</figure>" +
            '<pre><code class="go">' + escapa(c.codigo) + "</code></pre>" +
          "</article>";
      });
      html += "</div>";
    });
    lista.innerHTML = html;
    indice.innerHTML = htmlIndice;
  }

  // ---- Tema ----
  function aplicaTema(tema, animar) {
    raiz.dataset.tema = tema;
    document.querySelectorAll("img[data-escuro]").forEach(function (img) {
      img.src = tema === "escuro" ? img.dataset.escuro : img.dataset.claro;
    });
    var botao = document.querySelector(".tema");
    botao.setAttribute("aria-pressed", String(tema === "escuro"));
    botao.querySelector(".tema-rotulo").textContent = tema === "escuro" ? "Claro" : "Escuro";
    if (animar) {
      raiz.classList.remove("trocando");
      void raiz.offsetWidth;
      raiz.classList.add("trocando");
      setTimeout(function () { raiz.classList.remove("trocando"); }, 600);
    }
  }

  // ---- Copiar ----
  function ligaCopiar() {
    document.querySelectorAll("pre").forEach(function (pre) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "copiar";
      b.textContent = "Copiar";
      pre.appendChild(b);
    });
    document.addEventListener("click", function (ev) {
      var b = ev.target.closest(".copiar");
      if (!b) return;
      var dono = b.closest("[data-copiar]");
      var texto = dono ? dono.dataset.copiar : b.parentElement.querySelector("code").textContent;
      navigator.clipboard.writeText(texto).then(function () {
        b.textContent = "Copiado";
        setTimeout(function () { b.textContent = "Copiar"; }, 1400);
      });
    });
  }

  // ---- Índice: item da seção visível vira a pílula ----
  function ligaIndice() {
    var links = Array.prototype.slice.call(document.querySelectorAll(".indice a"));
    var alvos = links.map(function (a) { return document.querySelector(a.getAttribute("href")); });
    var atual = null;
    function marca() {
      var limite = window.innerHeight * 0.3, escolhido = -1;
      alvos.forEach(function (el, i) {
        if (el && el.getBoundingClientRect().top <= limite) escolhido = i;
      });
      var novo = links[escolhido] || null;
      if (novo === atual) return;
      if (atual) atual.removeAttribute("aria-current");
      atual = novo;
      if (atual) atual.setAttribute("aria-current", "true");
    }
    window.addEventListener("scroll", marca, { passive: true });
    marca();

    var menu = document.querySelector(".menu");
    var indice = document.getElementById("indice");
    menu.addEventListener("click", function () {
      var aberto = indice.classList.toggle("aberto");
      menu.setAttribute("aria-expanded", String(aberto));
    });
    indice.addEventListener("click", function (ev) {
      if (ev.target.closest("a")) {
        indice.classList.remove("aberto");
        menu.setAttribute("aria-expanded", "false");
      }
    });
  }

  montaComponentes();
  document.querySelectorAll("pre code.go").forEach(function (el) {
    el.innerHTML = destacaGo(el.textContent);
  });
  ligaCopiar();
  ligaIndice();
  aplicaTema(raiz.dataset.tema, false);
  document.querySelector(".tema").addEventListener("click", function () {
    var novo = raiz.dataset.tema === "escuro" ? "claro" : "escuro";
    localStorage.setItem("juigo-tema", novo);
    aplicaTema(novo, true);
  });
})();
