#!/usr/bin/env python3

import json
import os
import argparse
from collections import defaultdict

def get_all_keys(obj, prefix=''):
    """递归获取所有键，处理嵌套的情况"""
    keys = set()
    for key, value in obj.items():
        full_key = f"{prefix}.{key}" if prefix else key
        if isinstance(value, dict):
            keys.update(get_all_keys(value, full_key))
        else:
            keys.add(full_key)
    return keys

def get_value_by_path(obj, path):
    """根据路径获取嵌套字典中的值"""
    current = obj
    for part in path.split('.'):
        if part not in current:
            return None
        current = current[part]
        if isinstance(current, dict):
            return None
    return current

def set_value_by_path(obj, path, value):
    """根据路径在嵌套字典中设置值"""
    current = obj
    parts = path.split('.')
    for part in parts[:-1]:
        if part not in current:
            current[part] = {}
        current = current[part]
    current[parts[-1]] = value

def load_json_file(file_path):
    """加载JSON文件，返回字典"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading {file_path}: {e}")
        return {}

def save_json_file(file_path, data):
    """保存JSON文件"""
    try:
        with open(file_path, 'w', encoding='utf-8') as f:
            json.dump(data, f, ensure_ascii=False, indent=4)
        return True
    except Exception as e:
        print(f"Error saving {file_path}: {e}")
        return False

def find_missing_keys(base_keys, target_data):
    """查找目标数据中缺失的键"""
    target_keys = get_all_keys(target_data)
    missing_keys = base_keys - target_keys
    return sorted(missing_keys)

def check_missing_keys_only(locale_dir, base_file='en.json'):
    """仅检查缺失的键，不做修改"""
    base_path = os.path.join(locale_dir, base_file)
    if not os.path.exists(base_path):
        print(f"Base file not found: {base_path}")
        return
    
    base_data = load_json_file(base_path)
    if not base_data:
        return
    
    base_keys = get_all_keys(base_data)
    print(f"Base file ({base_file}) contains {len(base_keys)} keys\n")
    
    total_missing = 0
    for filename in sorted(os.listdir(locale_dir)):
        if not filename.endswith('.json') or filename == base_file:
            continue
        
        file_path = os.path.join(locale_dir, filename)
        target_data = load_json_file(file_path)
        if not target_data:
            continue
        
        missing_keys = find_missing_keys(base_keys, target_data)
        if missing_keys:
            print(f"{filename}: {len(missing_keys)} missing keys")
            total_missing += len(missing_keys)
        else:
            print(f"{filename}: ✓ Complete")
    
    print(f"\nTotal missing keys across all files: {total_missing}")

def add_missing_keys(locale_dir, base_file='en.json', placeholder_template='[TO TRANSLATE] {}', target_languages=None):
    """添加缺失的键到指定语言文件"""
    base_path = os.path.join(locale_dir, base_file)
    if not os.path.exists(base_path):
        print(f"Base file not found: {base_path}")
        return
    
    base_data = load_json_file(base_path)
    if not base_data:
        return
    
    base_keys = get_all_keys(base_data)
    print(f"Base file ({base_file}) contains {len(base_keys)} keys\n")
    
    success_count = 0
    total_count = 0
    
    for filename in sorted(os.listdir(locale_dir)):
        if not filename.endswith('.json') or filename == base_file:
            continue
        
        # 如果指定了目标语言，只处理指定的语言
        if target_languages:
            lang_code = filename[:-5]  # 移除.json后缀
            if lang_code not in target_languages:
                continue
        
        file_path = os.path.join(locale_dir, filename)
        print(f"Processing {filename}...")
        
        target_data = load_json_file(file_path)
        if not target_data:
            continue
        
        missing_keys = find_missing_keys(base_keys, target_data)
        
        if missing_keys:
            print(f"  Found {len(missing_keys)} missing keys")
            
            # 添加缺失的键
            added_count = 0
            for key in missing_keys:
                base_value = get_value_by_path(base_data, key)
                if base_value is not None:
                    placeholder_value = placeholder_template.format(base_value)
                    set_value_by_path(target_data, key, placeholder_value)
                    added_count += 1
            
            if added_count > 0 and save_json_file(file_path, target_data):
                print(f"  ✓ Added {added_count} missing keys")
                success_count += 1
            else:
                print(f"  ✗ Failed to save file")
        else:
            print(f"  ✓ No missing keys")
            success_count += 1
        
        total_count += 1
    
    print(f"\nSummary:")
    print(f"Processed {total_count} language files")
    print(f"Successfully updated {success_count} files")

def main():
    parser = argparse.ArgumentParser(description='Manage missing i18n translation keys')
    parser.add_argument('--action', choices=['check', 'add'], default='add',
                        help='Action to perform: check (only show missing keys) or add (add missing keys)')
    parser.add_argument('--base-file', default='en.json',
                        help='Base language file to use as reference (default: en.json)')
    parser.add_argument('--placeholder', default='[TO TRANSLATE] {}',
                        help='Placeholder template for missing translations (default: "[TO TRANSLATE] {}")')
    parser.add_argument('--languages', nargs='*',
                        help='Specific language codes to process (e.g., zh-cn fr de). If not specified, process all languages')
    parser.add_argument('--locale-dir',
                        help='Path to locale directory (default: ../../client/src/__locales relative to script)')
    
    args = parser.parse_args()
    
    # 确定locale目录
    if args.locale_dir:
        locale_dir = args.locale_dir
    else:
        script_dir = os.path.dirname(os.path.abspath(__file__))
        locale_dir = os.path.join(script_dir, '../../client/src/__locales')
    
    if not os.path.exists(locale_dir):
        print(f"Locale directory not found: {locale_dir}")
        return 1
    
    if args.action == 'check':
        check_missing_keys_only(locale_dir, args.base_file)
    else:
        add_missing_keys(locale_dir, args.base_file, args.placeholder, args.languages)
    
    return 0

if __name__ == "__main__":
    exit(main())
