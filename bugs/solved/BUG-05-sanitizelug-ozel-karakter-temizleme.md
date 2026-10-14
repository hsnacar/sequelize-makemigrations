# BUG-05: `sanitizeSlug` — Özel karakterleri temizlemiyor, geçersiz dosya adı üretiliyor

## Kritiklik: YÜKSEK

## Dosya
- `adapters/generator/v6/generator.go` satır 148-153
- `adapters/generator/v7/generator.go` satır 149-154

## Mevcut Kod

```go
func sanitizeSlug(name string) string {
    name = strings.TrimSpace(strings.ToLower(name))
    name = strings.ReplaceAll(name, " ", "-")
    name = strings.ReplaceAll(name, "_", "-")
    return name
}
```

## Bug Açıklaması

`sanitizeSlug` fonksiyonu sadece boşlukları ve alt çizgileri çeviriyor. Özel karakterleri temizlemiyor:

- `add user & bio` → `add-user-&-bio` → `add-user-&-bio.js`
- `create#index` → `create#index.js`
- `user+profile` → `user+profile.js`

Bu karakterler dosya adlarında geçersizdir veya shell'de sorun çıkarır.

## Etki

- `seq-migrate make -n "add user & bio"` çalıştırılırsa → `add-user-&-bio.js` dosyası oluşur.
- Bash/Linux'ta `&` karakteri arka plan işlemi başlatır → Dosya oluşturma başarısız olabilir.
- Windows'ta `#`, `@`, `+` karakterleri dosya adında sorun yaratır.

## Beklenen Davranış

Tüm özel karakterler temizlenmeli veya `-` ile değiştirilmeli:

```go
func sanitizeSlug(name string) string {
    name = strings.TrimSpace(strings.ToLower(name))
    name = strings.ReplaceAll(name, " ", "-")
    name = strings.ReplaceAll(name, "_", "-")
    // Özel karakterleri temizle
    var result strings.Builder
    for _, r := range name {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
            result.WriteRune(r)
        }
    }
    // Çoklu '-' tek '-'e indir
    slug := result.String()
    for strings.Contains(slug, "--") {
        slug = strings.ReplaceAll(slug, "--", "-")
    }
    return strings.Trim(slug, "-")
}
```

## Test Senaryosu

1. `seq-migrate make -n "add user & bio"` çalıştır
2. **Beklenen:** `add-user-bio.js` dosyası oluşmalı
3. **Mevcut davranış:** `add-user-&-bio.js` → Shell hatası veya geçersiz dosya adı
