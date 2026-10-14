# BUG-04: `RepairMissingDown` — Son `};` araması dosyayı bozabilir

## Kritiklik: YÜKSEK

## Dosya
`core/repair/repair.go` satır 90-96

## Mevcut Kod

```go
lastClosingIndex := strings.LastIndex(content, "};")
if lastClosingIndex == -1 {
    return fmt.Errorf("could not find closing '};' in %s", filePath)
}

repaired := content[:lastClosingIndex] + stub
return os.WriteFile(filePath, []byte(repaired), 0644)
```

`stub` değişkeni:
```go
stub := `
  async down(queryInterface, Sequelize) {
    // TODO: Add rollback operations
  }
};`
```

## Bug Açıklaması

`strings.LastIndex(content, "};")` dosyadaki **son** `};` arar. Ancak migration dosyasının string veya comment'lerinde `};` geçiyorsa:

```javascript
await queryInterface.sequelize.query("SELECT * FROM users WHERE data LIKE '};'");
```

Bu durumda `LastIndex` **string'in içindeki** `};`'i bulur ve down handler'ı string'in ortasına ekler → **sözdizimi hatası**.

## Etki

- `seq-migrate repair --fix-down` çalıştırıldığında migration dosyası bozulur.
- Bozulan migration artık `require` edilemez.
- Özellikle raw query kullanan migration'larda yaygın bir risk.

## Beklenen Davranış

Down handler'ı güvenli bir şekilde eklemek için:
1. Regex ile `module.exports` yapısını tanımalı
2. Sadece **en dış** `};` (kapanış bracket'ı) aramalı
3. Veya AST tabanlı bir parser kullanmalı

## Önerilen Düzeltme (Regex yaklaşımı)

```go
func RepairMissingDown(filePath string) error {
    contentBytes, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }
    content := string(contentBytes)
    if hasDownRegex.MatchString(content) {
        return nil // Zaten down var
    }

    // Regex: sadece başında boşluk olmayan (en dış seviye) "};"
    reClosing := regexp.MustCompile(`(?m)^\};`)
    matches := reClosing.FindAllStringIndex(content, -1)
    if len(matches) == 0 {
        return fmt.Errorf("could not find closing '};' in %s", filePath)
    }

    // En dış kapanış: en sonda olan
    lastMatch := matches[len(matches)-1]
    insertPos := lastMatch[0]

    repaired := content[:insertPos] +
        "\n  async down(queryInterface, Sequelize) {\n    // TODO: Add rollback operations\n  }\n" +
        content[insertPos:]

    return os.WriteFile(filePath, []byte(repaired), 0644)
}
```

## Test Senaryosu

1. Migration dosyası içinde: `await queryInterface.sequelize.query("SELECT '};' as test");`
2. `seq-migrate repair --fix-down` çalıştır
3. **Beklenen:** Down handler güvenli şekilde eklenmeli
4. **Mevcut davranış:** String içindeki `};` bulunur → Dosya bozulur
