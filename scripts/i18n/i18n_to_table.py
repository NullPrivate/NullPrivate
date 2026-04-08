#!/usr/bin/env python3

import json
import os
from collections import defaultdict

def read_locale_files(locale_dir):
    """读取所有语言文件并返回合并的数据结构"""
    all_keys = set()
    translations = {}
    
    # 读取所有语言文件
    for filename in os.listdir(locale_dir):
        if not filename.endswith('.json'):
            continue
            
        lang = filename[:-5]  # 移除.json后缀
        file_path = os.path.join(locale_dir, filename)
        
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
            translations[lang] = data
            all_keys.update(get_all_keys(data))
            
    return all_keys, translations

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
            return ''
        current = current[part]
        if not isinstance(current, dict):
            return current
    return current if not isinstance(current, dict) else ''

def create_markdown_table(all_keys, translations):
    """创建markdown格式的表格"""
    # 按字母顺序排序语言和键
    languages = sorted(translations.keys())
    sorted_keys = sorted(all_keys)
    
    # 创建表头
    header = "| Key | " + " | ".join(languages) + " |"
    separator = "|---" + "|---" * len(languages) + "|"
    
    # 创建表格内容
    rows = []
    for key in sorted_keys:
        row_values = [key]
        for lang in languages:
            value = get_value_by_path(translations[lang], key)
            # 处理markdown表格中的特殊字符
            value = str(value).replace('|', '\\|').replace('\n', '<br>')
            row_values.append(value)
        rows.append("| " + " | ".join(row_values) + " |")
    
    return "\n".join([header, separator] + rows)

def main():
    # 指定i18n文件目录
    locale_dir = "../../client/src/__locales"
    script_dir = os.path.dirname(os.path.abspath(__file__))
    locale_dir = os.path.join(script_dir, locale_dir)
    
    # 读取所有语言文件
    all_keys, translations = read_locale_files(locale_dir)
    
    # 生成markdown表格
    table = create_markdown_table(all_keys, translations)
    
    # 将结果写入文件
    output_file = os.path.join(script_dir, "i18n_table.md")
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(table)
    
    print(f"Markdown table has been generated and saved to {output_file}")

if __name__ == "__main__":
    main()