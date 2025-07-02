#!/usr/bin/env python3

import json
import os
import argparse
from collections import defaultdict, OrderedDict

def get_all_keys(obj, prefix=''):
    """递归获取所有键的列表"""
    keys = []
    for key, value in obj.items():
        full_key = f"{prefix}.{key}" if prefix else key
        if isinstance(value, dict):
            keys.extend(get_all_keys(value, full_key))
        else:
            keys.append(full_key)
    return keys

def parse_markdown_table(file_path):
    """解析markdown表格文件，返回语言数据"""
    languages = []
    translations = defaultdict(dict)
    
    with open(file_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    
    # 跳过可能的文件头部注释
    for i, line in enumerate(lines):
        if line.startswith('| Key |'):
            header_line = line
            separator_line = lines[i + 1]
            content_lines = lines[i + 2:]
            break
    
    # 解析表头获取语言列表
    headers = [h.strip() for h in header_line.split('|')]
    languages = headers[2:-1]  # 跳过'Key'列和空列
    
    # 解析内容行
    for line in content_lines:
        if not line.strip() or not line.startswith('|'):
            continue
            
        cols = [col.strip() for col in line.split('|')[1:-1]]
        if len(cols) < 2:
            continue
            
        key = cols[0]
        for i, lang in enumerate(languages):
            if i + 1 < len(cols):
                value = cols[i + 1]
                if value:  # 只保存非空的翻译
                    # 处理markdown表格中的特殊字符
                    value = value.replace('\\|', '|')
                    value = value.replace('<br>', '\n')
                    translations[lang][key] = value

    return languages, translations

def create_nested_dict(flat_dict):
    """将扁平的键值对转换为嵌套的字典结构"""
    nested = {}
    for key, value in flat_dict.items():
        parts = key.split('.')
        current = nested
        for part in parts[:-1]:
            current = current.setdefault(part, {})
        current[parts[-1]] = value
    return nested

def get_original_key_order(file_path):
    """获取原始JSON文件的key顺序"""
    if not os.path.exists(file_path):
        return []
    
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            original_data = json.load(f, object_pairs_hook=OrderedDict)
        return get_all_keys_ordered(original_data)
    except Exception as e:
        print(f"Warning: Could not read original file {file_path}: {e}")
        return []

def get_all_keys_ordered(obj, prefix=''):
    """递归获取所有键的有序列表"""
    keys = []
    for key, value in obj.items():
        full_key = f"{prefix}.{key}" if prefix else key
        if isinstance(value, dict):
            keys.extend(get_all_keys_ordered(value, full_key))
        else:
            keys.append(full_key)
    return keys

def create_nested_dict_ordered(flat_dict, key_order):
    """根据指定的key顺序创建嵌套的有序字典结构"""
    nested = OrderedDict()
    
    # 首先按照原有顺序添加已存在的key
    for key in key_order:
        if key in flat_dict:
            parts = key.split('.')
            current = nested
            for part in parts[:-1]:
                if part not in current:
                    current[part] = OrderedDict()
                current = current[part]
            current[parts[-1]] = flat_dict[key]
    
    # 然后添加新的key（如果有的话）
    for key, value in flat_dict.items():
        if key not in key_order:
            parts = key.split('.')
            current = nested
            for part in parts[:-1]:
                if part not in current:
                    current[part] = OrderedDict()
                current = current[part]
            current[parts[-1]] = value
    
    return nested

def merge_translations(original_data, new_translations, table_keys):
    """将新翻译合并到原数据中，删除不在表格中的key"""
    def set_nested_value(obj, path, value):
        """在嵌套字典中设置值"""
        parts = path.split('.')
        current = obj
        for part in parts[:-1]:
            if part not in current:
                current[part] = {}
            current = current[part]
        current[parts[-1]] = value
    
    def delete_nested_value(obj, path):
        """删除嵌套字典中的值"""
        parts = path.split('.')
        current = obj
        for part in parts[:-1]:
            if part not in current:
                return  # 路径不存在，无需删除
            current = current[part]
        if parts[-1] in current:
            del current[parts[-1]]
    
    def clean_empty_dicts(obj):
        """递归清理空的字典"""
        if not isinstance(obj, dict):
            return obj
        
        # 先递归清理子字典
        for key, value in list(obj.items()):
            if isinstance(value, dict):
                cleaned_value = clean_empty_dicts(value)
                if not cleaned_value:  # 如果子字典为空，删除它
                    del obj[key]
                else:
                    obj[key] = cleaned_value
        
        return obj
    
    # 复制原数据
    merged_data = json.loads(json.dumps(original_data))  # 深拷贝
    
    # 获取原数据中的所有key
    original_keys = set(get_all_keys(original_data))
    
    # 删除不在表格中的key
    keys_to_delete = original_keys - set(table_keys)
    for key in keys_to_delete:
        delete_nested_value(merged_data, key)
    
    # 更新新翻译中的值
    for key, value in new_translations.items():
        if value:  # 只更新非空值
            set_nested_value(merged_data, key, value)
    
    # 清理空的字典
    merged_data = clean_empty_dicts(merged_data)
    
    return merged_data

def save_language_files(translations, output_dir, table_keys, target_languages=None):
    """保存各个语言的JSON文件，保持原有的key顺序，合并而不是覆盖，删除不在表格中的key"""
    os.makedirs(output_dir, exist_ok=True)
    
    saved_count = 0
    for lang, trans in translations.items():
        # 如果指定了目标语言，只处理指定的语言
        if target_languages and lang not in target_languages:
            print(f"Skipping {lang}.json (not in target languages)")
            continue
            
        output_file = os.path.join(output_dir, f'{lang}.json')
        
        # 读取原始文件
        try:
            with open(output_file, 'r', encoding='utf-8') as f:
                original_data = json.load(f)
            original_key_count = len(get_all_keys(original_data))
            print(f"Loaded existing {lang}.json ({original_key_count} existing keys)")
        except (FileNotFoundError, json.JSONDecodeError) as e:
            print(f"Could not load existing {lang}.json: {e}")
            # 如果无法读取原文件，使用空字典
            original_data = {}
            original_key_count = 0
        
        # 合并翻译
        merged_data = merge_translations(original_data, trans, table_keys)
        updated_keys = len([k for k, v in trans.items() if v])  # 统计非空的更新
        final_key_count = len(get_all_keys(merged_data))
        deleted_keys = original_key_count - final_key_count + updated_keys
        
        print(f"Merging {updated_keys} translations into {lang}.json")
        if deleted_keys > 0:
            print(f"Deleted {deleted_keys} keys not found in table")
        
        # 保存为JSON文件
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(merged_data, f, ensure_ascii=False, indent=4)
        
        saved_count += 1
    
    return saved_count

def main():
    parser = argparse.ArgumentParser(description='Convert markdown table to i18n JSON files')
    parser.add_argument('--languages', nargs='*',
                        help='Specific language codes to process (e.g., zh-cn fr de). If not specified, process all languages')
    parser.add_argument('--input-file', 
                        help='Input markdown table file path (default: ./i18n_table.md)')
    parser.add_argument('--output-dir',
                        help='Output directory for JSON files (default: ../../client/src/__locales)')
    
    args = parser.parse_args()
    
    script_dir = os.path.dirname(os.path.abspath(__file__))
    
    # 确定输入文件路径
    if args.input_file:
        input_file = args.input_file
    else:
        input_file = os.path.join(script_dir, 'i18n_table.md')
    
    # 确定输出目录
    if args.output_dir:
        output_dir = args.output_dir
    else:
        output_dir = os.path.join(script_dir, '../../client/src/__locales')
    
    if not os.path.exists(input_file):
        print(f"Error: Input file not found: {input_file}")
        return 1
    
    print(f"Reading from: {input_file}")
    print(f"Output directory: {output_dir}")
    
    if args.languages:
        print(f"Target languages: {', '.join(args.languages)}")
    else:
        print("Processing all languages found in the table")
    
    languages, translations = parse_markdown_table(input_file)
    
    # 获取表格中的所有key
    table_keys = set()
    for lang_translations in translations.values():
        table_keys.update(lang_translations.keys())
    table_keys = list(table_keys)
    
    print(f"Found {len(table_keys)} unique keys in table")
    
    saved_count = save_language_files(translations, output_dir, table_keys, args.languages)
    
    if args.languages:
        total_languages = len([lang for lang in languages if lang in args.languages])
        print(f"\nSuccessfully generated {saved_count} of {total_languages} requested language files")
    else:
        print(f"\nSuccessfully generated {saved_count} of {len(languages)} language files")
    
    print(f"Files have been saved to: {output_dir}")
    
    return 0

if __name__ == "__main__":
    exit(main())