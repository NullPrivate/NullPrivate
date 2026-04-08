# i18n Translation Management Scripts

This directory contains scripts for managing internationalization (i18n) translation files.

## Scripts Overview

### 1. `i18n_to_table.py`
Converts JSON translation files to a markdown table format for easy editing.

**Usage:**
```bash
./i18n_to_table.py
```

**Output:** `i18n_table.md` - A markdown table containing all translations

### 2. `table_to_i18n.py`
Converts a markdown table back to JSON translation files while preserving the original key order.

**Usage:**
```bash
# Convert all languages from the table
./table_to_i18n.py

# Convert only specific languages
./table_to_i18n.py --languages zh-cn zh-hk fr

# Use custom input file and output directory
./table_to_i18n.py --input-file custom_table.md --output-dir /path/to/locales
```

**Options:**
- `--languages [LANGUAGES ...]`: Specific language codes to process (e.g., zh-cn fr de)
- `--input-file INPUT_FILE`: Input markdown table file path (default: ./i18n_table.md)
- `--output-dir OUTPUT_DIR`: Output directory for JSON files (default: ../../client/src/__locales)

**Features:**
- Preserves the original key order from existing JSON files
- Supports selective language processing
- Maintains JSON formatting with 4-space indentation
- **Merges translations** instead of overwriting - preserves keys that exist only in JSON files
- Only updates keys that are present in the markdown table

**Input:** `i18n_table.md` - A markdown table with translations
**Output:** Updated JSON files in the specified directory

### 3. `add_missing_keys.py`
Simple script to detect and add missing translation keys to language files.

**Usage:**
```bash
./add_missing_keys.py
```

**Features:**
- Uses `en.json` as the base reference file
- Adds missing keys with `[TO TRANSLATE]` prefix
- Processes all language files automatically

### 4. `manage_missing_keys.py` (Recommended)
Advanced script for managing missing translation keys with more options.

**Usage:**

Check for missing keys without making changes:
```bash
./manage_missing_keys.py --action check
```

Add missing keys to all language files:
```bash
./manage_missing_keys.py --action add
```

Add missing keys only to specific languages:
```bash
./manage_missing_keys.py --action add --languages zh-cn fr de
```

Use a custom placeholder template:
```bash
./manage_missing_keys.py --action add --placeholder "TODO: {}"
```

Use a different base language file:
```bash
./manage_missing_keys.py --action add --base-file de.json
```

**Options:**
- `--action {check,add}`: Choose whether to check or add missing keys
- `--base-file BASE_FILE`: Base language file to use as reference (default: en.json)
- `--placeholder PLACEHOLDER`: Template for missing translations (default: "[TO TRANSLATE] {}")
- `--languages [LANGUAGES ...]`: Specific language codes to process
- `--locale-dir LOCALE_DIR`: Custom path to locale directory

## Workflow Examples

### Adding New Translation Keys

1. **Add new keys to the base language file** (usually `en.json`)
2. **Check which files need updates:**
   ```bash
   ./manage_missing_keys.py --action check
   ```
3. **Add missing keys to all language files:**
   ```bash
   ./manage_missing_keys.py --action add
   ```
4. **Review and translate the added keys** (look for `[TO TRANSLATE]` prefix)

### Working with Specific Languages

1. **Check missing keys for specific languages:**
   ```bash
   ./manage_missing_keys.py --action check
   ```
2. **Add missing keys only to Chinese and French translations:**
   ```bash
   ./manage_missing_keys.py --action add --languages zh-cn fr
   ```

### Bulk Translation Editing

1. **Export to markdown table:**
   ```bash
   ./i18n_to_table.py
   ```
2. **Edit `i18n_table.md` in your preferred editor**
3. **Import back to JSON files (all languages):**
   ```bash
   ./table_to_i18n.py
   ```
4. **Or import only specific languages:**
   ```bash
   ./table_to_i18n.py --languages zh-cn fr de
   ```

### Safe Translation Updates

When using `table_to_i18n.py`, the script **safely merges** translations:

- ✅ **Keys in both JSON and table**: Updated with table values
- ✅ **Keys only in JSON**: Preserved unchanged  
- ✅ **Keys only in table**: Added to JSON
- ✅ **Original key order**: Maintained

This means you can safely update translations without worrying about losing keys that exist only in your JSON files but not in the markdown table.

### Testing Translation Changes

1. **Export current translations to table:**
   ```bash
   ./i18n_to_table.py
   ```
2. **Make changes to specific languages in the table**
3. **Import only the changed languages for testing:**
   ```bash
   ./table_to_i18n.py --languages zh-cn
   ```
4. **Test your application with the updated translations**
5. **Import remaining languages when satisfied:**
   ```bash
   ./table_to_i18n.py --languages fr de es
   ```

## File Structure

```
scripts/i18n/
├── add_missing_keys.py         # Simple missing key detection
├── manage_missing_keys.py      # Advanced missing key management
├── i18n_to_table.py           # JSON to markdown table
├── table_to_i18n.py           # Markdown table to JSON
├── i18n_table.md              # Generated markdown table (if exists)
└── README.md                  # This file
```

## Notes

- All scripts assume the locale directory is at `../../client/src/__locales/` relative to the script location
- The base language file (usually `en.json`) should contain all possible translation keys
- Missing keys are added with a `[TO TRANSLATE]` prefix to make them easy to identify
- JSON files are formatted with 4-space indentation for consistency
- Scripts handle nested translation keys (e.g., `form.error.required`)
