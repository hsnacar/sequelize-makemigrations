# BUG-03: `formatColumnAttributes` — Karmaşık DefaultValue tipleri için bozuk JavaScript üretiliyor

## Kritiklik: KRİTİK

## Dosya
- `adapters/generator/v6/generator.go` satır 232-241
- `adapters/generator/v7/generator.go` satır 242-250

## Mevcut Kod (v6)

```go
if col.DefaultValue != nil {
    switch v := col.DefaultValue.(type) {
    case string:
        sb.WriteString(fmt.Sprintf("      defaultValue: '%s',\n", v))
    case bool:
        sb.WriteString(fmt.Sprintf("      defaultValue: %t,\n", v))
    default:
        sb.WriteString(fmt.Sprintf("      defaultValue: %v,\n", v))  // ← BUG
    }
}
```

Aynı kod v7 içinde var (formatColumnAttributes fonksiyonu, sadece formatDataType çağrısı farklı).

## Bug Açıklaması

`DefaultValue` şu tiplerde olduğunda `default` case'e düşer ve `%v` kullanılır:

| Tip | Go değeri | `%v` çıktısı | Geçerli JS mi? |
|-----|-----------|-------------|-----------------|
| `int` | `42` | `42` | ✅ (ama float64 olabilir) |
| `float64` | `3.14` | `3.14` | ✅ |
| `[]interface{}` | `["a","b"]` | `[a b]` | ❌ |
| `map[string]interface{}` | `{"k":"v"}` | `map[k:v]` | ❌ |
| `time.Time` | `time.Now()` | `2024-01-01 00:00:00 +0000 UTC` | ❌ |

JSON'dan gelen tüm numeric değerler Go'da `float64`'a unmarshal edilir. `case string` ve `case bool` hariç her şey `default` case'e düşer.

## Etki

Migration dosyası **sözdizimi hatalı** JavaScript kodu içerir:
- Array default: `defaultValue: [a b]` → JavaScript parse hatası
- Object default: `defaultValue: map[k:v]` → JavaScript parse hatası
- Bu migration `require` edildiğinde **SyntaxError** fırlatır → Verifier bile geçemez.

## Beklenen Davranış

Karmaşık tipler `json.Marshal` ile serialize edilmeli veya tip bazlı işlenmeli:

- Array/Object → JSON string
- Number → doğrudan yazılabilir
- `nil` → `null`

## Önerilen Düzeltme

```go
if col.DefaultValue != nil {
    switch v := col.DefaultValue.(type) {
    case string:
        sb.WriteString(fmt.Sprintf("      defaultValue: '%s',\n", v))
    case bool:
        sb.WriteString(fmt.Sprintf("      defaultValue: %t,\n", v))
    case float64:
        // JSON numbers are float64 in Go
        sb.WriteString(fmt.Sprintf("      defaultValue: %v,\n", v))
    case int:
        sb.WriteString(fmt.Sprintf("      defaultValue: %d,\n", v))
    default:
        // Complex types: arrays, objects, etc. → JSON marshal
        jsonBytes, err := json.Marshal(v)
        if err == nil {
            sb.WriteString(fmt.Sprintf("      defaultValue: %s,\n", string(jsonBytes)))
        }
    }
}
```

Aynı düzeltme hem `v6/generator.go` hem `v7/generator.go` için uygulanmalı.

## Test Senaryosu

1. Modelde bir sütun: `defaultValue: ['active', 'pending']` (ENUM array)
2. `seq-migrate make` çalıştır
3. Üretilen migration dosyasını incele
4. **Beklenen:** `defaultValue: ["active","pending"]`
5. **Mevcut davranış:** `defaultValue: [active pending]` → SyntaxError
