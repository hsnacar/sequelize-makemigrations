# BUG-07: `bridge.js` — `findModelsPath` klasör/dosya önceliği yanlış

## Kritiklik: YÜKSEK

## Dosya
`adapters/extractor/bridge.js` satır 115-131

## Mevcut Kod

```javascript
function findModelsPath(root) {
    const candidates = [
        path.join(root, 'models'),
        path.join(root, 'models', 'index.js'),
        path.join(root, 'models', 'index.ts'),
        path.join(root, 'src', 'models'),
        path.join(root, 'src', 'models', 'index.js'),
        path.join(root, 'src', 'models', 'index.ts')
    ];

    for (const candidate of candidates) {
        if (fs.existsSync(candidate)) {
            return candidate;
        }
    }
    return null;
}
```

## Bug Açıklaması

`fs.existsSync` hem dosya hem klasör için `true` döner. Bu şu sorunlara yol açar:

1. **`models` bir dosya ise** (örn. `models.js`), ilk kontrolde `true` döner ama `require()` klasör beklediği için hata verebilir.
2. **`models/` klasörü ve `models/index.js` dosyası ikisi de varsa**, önce `models` (klasör) bulunur → `require("models/")` çağrılır → Bu çalışabilir ama `models/index.js`'e göre farklı davranabilir.
3. **`src/models` öncelikli olmalı ama `src/models/index.js` bulunursa**, `require("src/models/")` yerine `require("src/models/index.js")` daha güvenli.

## Etki

- Beklenmedik `require()` hataları.
- Farklı projelerde tutarsız davranış.
- TypeScript projelerinde `.ts` uzantılı index dosyaları bulunamayabilir.

## Beklenen Davranış

1. Önce `index.js`/`index.ts` aranmalı (daha spesifik).
2. Sonra klasör varlığı kontrol edilmeli.
3. `statSync().isDirectory()` ile dosya/klasır ayrımı yapılmalı.

```javascript
function findModelsPath(root) {
    const candidates = [
        { path: path.join(root, 'models', 'index.js'), priority: 1 },
        { path: path.join(root, 'models', 'index.ts'), priority: 2 },
        { path: path.join(root, 'models'), priority: 3 },
        { path: path.join(root, 'src', 'models', 'index.js'), priority: 4 },
        { path: path.join(root, 'src', 'models', 'index.ts'), priority: 5 },
        { path: path.join(root, 'src', 'models'), priority: 6 },
    ];

    for (const { path: candidate } of candidates) {
        try {
            const stat = fs.statSync(candidate);
            if (stat.isFile() || stat.isDirectory()) {
                return candidate;
            }
        } catch (_) {}
    }
    return null;
}
```

## Test Senaryosu

1. Proje yapısı: `models/index.js` ve `models/` klasörü ikisi de var
2. `seq-migrate make` çalıştır
3. **Beklenen:** `models/index.js` bulunmalı
4. **Mevcut davranış:** `models` klasörü bulunur → `require("models/")` → farklı sonuç
