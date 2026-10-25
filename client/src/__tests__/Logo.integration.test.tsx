import React from 'react';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, unmountComponentAtNode } from 'react-dom';
import { act } from 'react-dom/test-utils';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { Logo } from '../components/ui/svg/logo';

describe('商标组件与页面资源集成', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        container = document.createElement('div');
        document.body.appendChild(container);
    });

    afterEach(() => {
        unmountComponentAtNode(container);
        container.remove();
    });

    it.each([
        ['index.html', 'header-brand-img'],
        ['login.html', 'h-6 login__logo'],
        ['install.html', 'setup__logo'],
    ])('%s 与商标组件使用同一图片并保留文字和尺寸', (page, className) => {
        act(() => {
            render(<Logo className={className} />, container);
        });

        const html = readFileSync(resolve('public', page), 'utf8');
        const document = new DOMParser().parseFromString(html, 'text/html');
        const iconPath = document.querySelector('link[rel="icon"]')?.getAttribute('href');
        const image = container.querySelector('image');

        expect(iconPath).toBe('assets/favicon.png');
        expect(image?.getAttribute('href')).toBe(iconPath);
        expect(image?.getAttribute('width')).toBe('40');
        expect(image?.getAttribute('height')).toBe('40');
        expect(container.querySelector('svg')?.getAttribute('viewBox')).toBe('0 0 160 40');
        expect(container.querySelector('text')?.textContent).toBe('宁屏');
        // 检查公共资源确实为 PNG，构建时由现有资源复制流程发布。
        const icon = readFileSync(resolve('public', iconPath!));
        expect([...icon.subarray(0, 8)]).toEqual([137, 80, 78, 71, 13, 10, 26, 10]);
    });
});