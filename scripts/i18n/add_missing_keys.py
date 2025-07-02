#!/usr/bin/env python3

import json
import os
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

def add_missing_keys_to_file(file_path, missing_keys, base_data):
    """将缺失的键添加到指定文件"""
    # 加载目标文件
    target_data = load_json_file(file_path)
    if not target_data:
        print(f"Failed to load {file_path}, skipping...")
        return False
    
    # 添加缺失的键
    added_count = 0
    for key in missing_keys:
        base_value = get_value_by_path(base_data, key)
        if base_value is not None:
            # 添加注释说明这是待翻译的键
            placeholder_value = f"[TO TRANSLATE] {base_value}"
            set_value_by_path(target_data, key, placeholder_value)
            added_count += 1
    
    if added_count > 0:
        # 保存文件
        if save_json_file(file_path, target_data):
            print(f"Added {added_count} missing keys to {os.path.basename(file_path)}")
            return True
        else:
            print(f"Failed to save {file_path}")
            return False
    else:
        print(f"No missing keys found in {os.path.basename(file_path)}")
        return True

def main():
    # 获取脚本目录和i18n文件目录
    script_dir = os.path.dirname(os.path.abspath(__file__))
    locale_dir = os.path.join(script_dir, '../../client/src/__locales')
    
    if not os.path.exists(locale_dir):
        print(f"Locale directory not found: {locale_dir}")
        return
    
    # 以英文文件作为基准
    base_file = os.path.join(locale_dir, 'en.json')
    if not os.path.exists(base_file):
        print(f"Base file not found: {base_file}")
        return
    
    # 加载基准文件
    base_data = load_json_file(base_file)
    if not base_data:
        print("Failed to load base file")
        return
    
    # 获取基准文件中的所有键
    base_keys = get_all_keys(base_data)
    print(f"Base file ({os.path.basename(base_file)}) contains {len(base_keys)} keys")
    
    # 处理所有其他语言文件
    success_count = 0
    total_count = 0
    
    for filename in sorted(os.listdir(locale_dir)):
        if not filename.endswith('.json') or filename == 'en.json':
            continue
        
        file_path = os.path.join(locale_dir, filename)
        print(f"\nProcessing {filename}...")
        
        # 加载目标文件
        target_data = load_json_file(file_path)
        if not target_data:
            continue
        
        # 查找缺失的键
        missing_keys = find_missing_keys(base_keys, target_data)
        
        if missing_keys:
            print(f"Found {len(missing_keys)} missing keys:")
            for key in missing_keys[:5]:  # 只显示前5个
                print(f"  - {key}")
            if len(missing_keys) > 5:
                print(f"  ... and {len(missing_keys) - 5} more")
            
            # 添加缺失的键
            if add_missing_keys_to_file(file_path, missing_keys, base_data):
                success_count += 1
        else:
            print(f"No missing keys found in {filename}")
            success_count += 1
        
        total_count += 1
    
    print(f"\nSummary:")
    print(f"Processed {total_count} language files")
    print(f"Successfully updated {success_count} files")
    
    if success_count < total_count:
        print(f"Failed to update {total_count - success_count} files")

if __name__ == "__main__":
    main()
